import { useTransport } from "@connectrpc/connect-query";
import { useQuery } from "@tanstack/react-query";
import { OWN_ENROLLMENTS_FOR_FILTER_QUERY_KEY } from "../api/queries";
import { createRpcOwnGradesSource } from "../api/rpc";

/** One option in the carrera (program) dropdown. */
export interface ProgramOption {
  /** UUID of the program, used as the filter value. */
  id: string;
  /** Display name of the program derived from grade data. */
  name: string;
}

/** Query result for carrera dropdown options. */
export interface UseOwnEnrollmentsForFilterResult {
  /** Unique program IDs from the student's enrollments. Names are empty until resolved from grade data. */
  programIds: string[];
  isLoading: boolean;
}

/**
 * Fetches the student's own enrollment program IDs via ListOwnEnrollments.
 * Returns de-duplicated program IDs for use as carrera filter options.
 * Requires `enrollment.view_own` permission (all students have it).
 */
export function useOwnEnrollmentsForFilter(): UseOwnEnrollmentsForFilterResult {
  const transport = useTransport();
  const source = createRpcOwnGradesSource(transport);

  const result = useQuery({
    queryKey: OWN_ENROLLMENTS_FOR_FILTER_QUERY_KEY,
    queryFn: () => source.listOwnEnrollments(),
    staleTime: 60_000,
  });

  const seen = new Set<string>();
  const programIds: string[] = [];
  for (const enrollment of result.data ?? []) {
    if (!seen.has(enrollment.programId)) {
      seen.add(enrollment.programId);
      programIds.push(enrollment.programId);
    }
  }

  return {
    programIds,
    isLoading: result.isLoading,
  };
}

/**
 * Derives carrera display options by correlating enrollment program IDs with
 * program names carried in loaded OwnGrade rows.
 * Programs whose names are not yet in the grade data show the UUID as a fallback.
 */
export function buildProgramOptions(
  programIds: string[],
  gradeNamesByProgramId: Map<string, string>,
): ProgramOption[] {
  return programIds.map((id) => ({
    id,
    name: gradeNamesByProgramId.get(id) ?? id,
  }));
}
