import { createClient } from "@connectrpc/connect";
import { useQuery, useTransport } from "@connectrpc/connect-query";
import { useCallback } from "react";
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
   * Merges fresh grade data for a single student row into the VM.
   * Called after a CodeAborted (optimistic lock conflict) refetch.
   * Returns a new map for the row without mutating state.
   */
  mergeRowGrades: (sectionEnrollmentId: string) => Promise<Map<string, CellVM>>;
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

  // Client for imperative refetch in mergeRowGrades
  const transport = useTransport();
  const gradesClient = createClient(GradesService, transport);

  const mergeRowGrades = useCallback(
    async (sectionEnrollmentId: string): Promise<Map<string, CellVM>> => {
      // Re-fetch all grades for the section and extract the row's fresh cells.
      const fresh = await gradesClient.listGradesForSection({
        sectionId,
        pageSize: GRADES_PAGE_SIZE,
        pageToken: "",
      });

      const freshCells = new Map<string, CellVM>();
      for (const ev of evaluations) {
        const grade = fresh.grades.find(
          (g) =>
            g.sectionEnrollmentId === sectionEnrollmentId &&
            g.evaluationId === ev.id,
        );
        freshCells.set(
          ev.id,
          grade
            ? {
                evaluationId: grade.evaluationId,
                value: grade.value,
                version: grade.version,
                gradeId: grade.id,
              }
            : {
                evaluationId: ev.id,
                value: "",
                version: 0,
                gradeId: "",
              },
        );
      }
      return freshCells;
    },
    [gradesClient, sectionId, evaluations],
  );

  return {
    evaluations,
    rows,
    isLoading,
    isError,
    mergeRowGrades,
  };
}
