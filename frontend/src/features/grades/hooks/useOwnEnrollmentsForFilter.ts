import { useTransport } from "@connectrpc/connect-query";
import { useQuery } from "@tanstack/react-query";
import { OWN_ENROLLMENTS_FOR_FILTER_QUERY_KEY } from "../api/queries";
import { createRpcOwnGradesSource } from "../api/rpc";

/** One option in the carrera (program) dropdown. */
export interface ProgramOption {
  /** UUID of the program, used as the filter value. */
  id: string;
  /** Display name of the program (carrera). */
  name: string;
}

/** Query result for carrera dropdown options. */
export interface UseOwnEnrollmentsForFilterResult {
  /** The student's distinct programs (carreras): id + display name. */
  programs: ProgramOption[];
  isLoading: boolean;
}

/**
 * Returns the student's distinct programs (carreras) for the carrera filter
 * dropdown, sourced from ListOwnEnrollments — each enrollment carries its
 * program id and name. Requires `enrollment.view_own`, which students hold.
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
  const programs: ProgramOption[] = [];
  for (const enrollment of result.data ?? []) {
    if (!seen.has(enrollment.programId)) {
      seen.add(enrollment.programId);
      programs.push({ id: enrollment.programId, name: enrollment.programName });
    }
  }

  return { programs, isLoading: result.isLoading };
}
