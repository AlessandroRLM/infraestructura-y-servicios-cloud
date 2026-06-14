import { queryOptions } from "@tanstack/react-query";
import type { ProgramCoursesSource } from "./rpc";

export const PROGRAM_COURSES_QUERY_KEY = (programId: string) =>
  ["catalog", "program-courses", programId] as const;

export function programCoursesQueryOptions(
  source: ProgramCoursesSource,
  programId: string,
) {
  return queryOptions({
    queryKey: PROGRAM_COURSES_QUERY_KEY(programId),
    queryFn: () => source.listProgramCourses(programId),
    staleTime: 30_000,
  });
}
