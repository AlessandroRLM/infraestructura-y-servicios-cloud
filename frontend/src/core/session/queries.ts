import { queryOptions } from "@tanstack/react-query";
import type { SessionSource } from "./source";

export const SESSION_QUERY_KEY = ["session"] as const;

export function bootstrapQueryOptions(source: SessionSource) {
  return queryOptions({
    queryKey: SESSION_QUERY_KEY,
    queryFn: () => source.getSession(),
    staleTime: Infinity,
  });
}
