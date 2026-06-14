import { useTransport } from "@connectrpc/connect-query";
import { useQuery } from "@tanstack/react-query";
import { programCoursesQueryOptions } from "../api/queries";
import { createRpcProgramCoursesSource } from "../api/rpc";

/**
 * Returns the ProgramCourse[] for a given program from ListProgramCourses.
 * Uses the injected-source seam for stub testability.
 */
export function useProgramCourses(programId: string) {
  const transport = useTransport();
  const source = createRpcProgramCoursesSource(transport);
  const result = useQuery(programCoursesQueryOptions(source, programId));
  return {
    programCourses: result.data ?? [],
    isPending: result.isPending,
    isError: result.isError,
    refetch: result.refetch,
  };
}
