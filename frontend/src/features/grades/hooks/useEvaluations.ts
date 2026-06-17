import { useQuery } from "@connectrpc/connect-query";
import type { Evaluation } from "@/gen/grades/v1/grades_pb";
import { GradesService } from "@/gen/grades/v1/grades_pb";

/** Result shape returned by useEvaluations. */
export interface UseEvaluationsResult {
  /** Evaluations for the selected course, sorted by position server-side. */
  evaluations: Evaluation[];
  /**
   * True while the initial query is loading and no data is available yet.
   *
   * NOTE (TanStack Query v5): `isPending` is also `true` when the query is
   * disabled (i.e. `courseId === ""`), because a disabled query is treated as
   * "pending with no data". Callers MUST guard on `courseId !== ""` before
   * treating this as an active loading state (SchemeManagementView does so via
   * the `schemeStateKnown` flag).
   */
  isPending: boolean;
  /** True if the query failed. */
  isError: boolean;
  /** Re-triggers the query, e.g. for a Retry action. */
  refetch: () => void;
}

/**
 * Fetches the evaluation scheme for a given course via ListEvaluations.
 * The query is disabled (no request issued) when courseId is an empty string.
 * An empty evaluations array means the course has no scheme yet.
 *
 * @param courseId - UUID of the course; pass "" to keep the query idle.
 */
export function useEvaluations(courseId: string): UseEvaluationsResult {
  const result = useQuery(
    GradesService.method.listEvaluations,
    { courseId },
    { enabled: courseId !== "" },
  );

  return {
    evaluations: result.data?.evaluations ?? [],
    isPending: result.isPending,
    isError: result.isError,
    refetch: result.refetch,
  };
}
