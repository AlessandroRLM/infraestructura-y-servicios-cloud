import { queryOptions } from "@tanstack/react-query";
import type { CoursesSource, ProgramsSource } from "./rpc";

export const PROGRAMS_QUERY_KEY = ["catalog", "programs"] as const;

export function programsQueryOptions(source: ProgramsSource) {
  return queryOptions({
    queryKey: PROGRAMS_QUERY_KEY,
    queryFn: () => source.listPrograms(),
    staleTime: 30_000,
  });
}

export const COURSES_QUERY_KEY = ["catalog", "courses"] as const;

export function coursesQueryOptions(source: CoursesSource) {
  return queryOptions({
    queryKey: COURSES_QUERY_KEY,
    queryFn: () => source.listCourses(),
    staleTime: 30_000,
  });
}
