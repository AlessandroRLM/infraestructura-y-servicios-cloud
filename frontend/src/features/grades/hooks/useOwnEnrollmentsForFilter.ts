import { useQuery } from "@connectrpc/connect-query";
import { EnrollmentService } from "@/gen/enrollment/v1/enrollment_pb";

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
 * Page size is 200 to cover the realistic max in a single call.
 */
export function useOwnEnrollmentsForFilter(): UseOwnEnrollmentsForFilterResult {
  const result = useQuery(
    EnrollmentService.method.listOwnEnrollments,
    { pageSize: 200, pageToken: "" },
    { staleTime: 60_000 },
  );

  const seen = new Set<string>();
  const programs: ProgramOption[] = [];
  for (const enrollment of result.data?.enrollments ?? []) {
    if (!seen.has(enrollment.programId)) {
      seen.add(enrollment.programId);
      programs.push({ id: enrollment.programId, name: enrollment.programName });
    }
  }

  return { programs, isLoading: result.isLoading };
}
