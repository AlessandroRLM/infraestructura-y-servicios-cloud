import { useQuery } from "@connectrpc/connect-query";
import { createConnectQueryKey } from "@connectrpc/connect-query-core";
import { useQueryClient } from "@tanstack/react-query";
import type { Evaluation } from "@/gen/grades/v1/grades_pb";
import { GradesService } from "@/gen/grades/v1/grades_pb";
import { ProfileService } from "@/gen/profiles/v1/profiles_pb";
import { SectionEnrollmentService } from "@/gen/section_enrollment/v1/section_enrollment_pb";

/** Per-cell grade state in the VM. */
export interface CellVM {
  /** Evaluation ID this cell belongs to. */
  evaluationId: string;
  /** Current grade value string (e.g. "5.5"), or "" if unrecorded. */
  value: string;
  /** Optimistic-lock version; 0 for unrecorded cells. */
  version: number;
  /** Grade record ID, or "" if unrecorded. */
  gradeId: string;
}

/** Per-row student data in the VM. */
export interface RowVM {
  /** SectionEnrollment ID — used as the row key. */
  sectionEnrollmentId: string;
  /** Raw student user ID — used for profile resolution + grade writes. */
  studentId: string;
  /** Resolved display name (given_names + last_name_paternal), or raw studentId as fallback. */
  displayName: string;
  /** Enrollment status: "in_progress", "withdrawn", "passed", "failed". */
  status: string;
  /** Ordered map of evaluationId → CellVM. */
  cells: Map<string, CellVM>;
}

/** Full grid view-model returned by useSectionGrid. */
export interface SectionGridVM {
  /** Ordered list of evaluations (columns). */
  evaluations: Evaluation[];
  /** Ordered list of student rows. */
  rows: RowVM[];
  /** True while any of the constituent queries are loading. */
  isLoading: boolean;
  /** True when one of the required queries failed. */
  isError: boolean;
  /**
   * Invalidates the listGradesForSection query cache entry for this section.
   * Called after a CodeAborted (optimistic lock conflict) so TanStack Query
   * re-fetches the authoritative server data. Rows update automatically via
   * the query subscription — no imperative fetch or local state merge needed.
   */
  refetchGrades: () => Promise<void>;
  /**
   * Invalidates all section queries (roster + grades + display names).
   * Used for section-scoped error recovery — refetches only this section's
   * data without reloading the page.
   */
  refetch: () => Promise<void>;
}

const ROSTER_PAGE_SIZE = 200;
const GRADES_PAGE_SIZE = 200;

/**
 * Composes four RPCs into a students × evaluations view-model for a given section.
 *
 * Constituent RPCs:
 *   1. ListSectionRosterForTeacher — roster (student_id + se_id + status)
 *   2. ListDisplayNamesByIDs — display names (gate: profile.view_names; falls back to studentId)
 *   3. ListGradesForSection — grade values + versions
 *   4. ListEvaluations — evaluation columns + weights
 *
 * The query is disabled (no requests issued) when sectionId is an empty string.
 *
 * @param sectionId - UUID of the selected section; pass "" to keep queries idle.
 * @param courseId - UUID of the course; required for ListEvaluations.
 */
export function useSectionGrid(
  sectionId: string,
  courseId: string,
): SectionGridVM {
  const enabled = sectionId !== "" && courseId !== "";

  // --- 1. Roster ---
  const rosterQuery = useQuery(
    SectionEnrollmentService.method.listSectionRosterForTeacher,
    { sectionId, pageSize: ROSTER_PAGE_SIZE, pageToken: "" },
    { enabled },
  );

  // --- 2. Display names ---
  const studentIds =
    rosterQuery.data?.sectionEnrollments.map((se) => se.studentId) ?? [];

  const displayNamesQuery = useQuery(
    ProfileService.method.listDisplayNamesByIDs,
    { userIds: studentIds },
    { enabled: enabled && studentIds.length > 0 },
  );

  // --- 3. Grades ---
  const gradesQuery = useQuery(
    GradesService.method.listGradesForSection,
    { sectionId, pageSize: GRADES_PAGE_SIZE, pageToken: "" },
    { enabled },
  );

  // --- 4. Evaluations (columns) ---
  const evaluationsQuery = useQuery(
    GradesService.method.listEvaluations,
    { courseId },
    { enabled },
  );

  const isLoading =
    rosterQuery.isLoading ||
    gradesQuery.isLoading ||
    evaluationsQuery.isLoading;

  const isError =
    rosterQuery.isError || gradesQuery.isError || evaluationsQuery.isError;

  // Build display-name lookup map (userId → displayName string)
  const nameMap = new Map<string, string>();
  if (displayNamesQuery.data) {
    for (const dn of displayNamesQuery.data.names) {
      nameMap.set(dn.userId, `${dn.givenNames} ${dn.lastNamePaternal}`);
    }
  }

  // Build grade lookup map (sectionEnrollmentId + evaluationId → CellVM)
  const gradeMap = new Map<string, CellVM>();
  if (gradesQuery.data) {
    for (const grade of gradesQuery.data.grades) {
      const key = `${grade.sectionEnrollmentId}::${grade.evaluationId}`;
      gradeMap.set(key, {
        evaluationId: grade.evaluationId,
        value: grade.value,
        version: grade.version,
        gradeId: grade.id,
      });
    }
  }

  const evaluations = evaluationsQuery.data?.evaluations ?? [];

  // Build rows
  const rows: RowVM[] = [];
  if (rosterQuery.data && !isLoading) {
    for (const se of rosterQuery.data.sectionEnrollments) {
      const cells = new Map<string, CellVM>();
      for (const ev of evaluations) {
        const key = `${se.id}::${ev.id}`;
        const existing = gradeMap.get(key);
        cells.set(
          ev.id,
          existing ?? {
            evaluationId: ev.id,
            value: "",
            version: 0,
            gradeId: "",
          },
        );
      }
      rows.push({
        sectionEnrollmentId: se.id,
        studentId: se.studentId,
        displayName: nameMap.get(se.studentId) ?? se.studentId,
        status: se.status,
        cells,
      });
    }
  }

  const queryClient = useQueryClient();

  // Invalidates the listGradesForSection cache entry so TanStack Query
  // triggers a background re-fetch. Rows update automatically via the
  // query subscription — no imperative client or local state merge.
  const refetchGrades = (): Promise<void> =>
    queryClient.invalidateQueries({
      queryKey: createConnectQueryKey({
        schema: GradesService.method.listGradesForSection,
        input: { sectionId, pageSize: GRADES_PAGE_SIZE, pageToken: "" },
        cardinality: "finite",
      }),
    });

  // Invalidates all three section queries (roster + grades + display names)
  // for section-scoped error recovery. Mirrors what happens on a full reload
  // but scoped to this section — does not touch unrelated cache entries.
  const refetch = (): Promise<void> =>
    Promise.all([
      queryClient.invalidateQueries({
        queryKey: createConnectQueryKey({
          schema: SectionEnrollmentService.method.listSectionRosterForTeacher,
          input: { sectionId, pageSize: ROSTER_PAGE_SIZE, pageToken: "" },
          cardinality: "finite",
        }),
      }),
      queryClient.invalidateQueries({
        queryKey: createConnectQueryKey({
          schema: GradesService.method.listGradesForSection,
          input: { sectionId, pageSize: GRADES_PAGE_SIZE, pageToken: "" },
          cardinality: "finite",
        }),
      }),
      queryClient.invalidateQueries({
        queryKey: createConnectQueryKey({
          schema: ProfileService.method.listDisplayNamesByIDs,
          input: { userIds: studentIds },
          cardinality: "finite",
        }),
      }),
    ]).then(() => undefined);

  return {
    evaluations,
    rows,
    isLoading,
    isError,
    refetchGrades,
    refetch,
  };
}
