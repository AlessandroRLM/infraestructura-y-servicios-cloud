import { queryOptions } from "@tanstack/react-query";
import type { ProgramsSource } from "./rpc";

export const PROGRAMS_QUERY_KEY = ["catalog", "programs"] as const;

export function programsQueryOptions(source: ProgramsSource) {
  return queryOptions({
    queryKey: PROGRAMS_QUERY_KEY,
    queryFn: () => source.listPrograms(),
    staleTime: 30_000,
  });
}
