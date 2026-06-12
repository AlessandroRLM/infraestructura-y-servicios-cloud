import { queryOptions } from "@tanstack/react-query";
import type { SessionSource } from "../types";

// The query key is the cache identity — every caller must share the same source
// instance, otherwise the first-registered queryFn wins silently.
export const SESSION_QUERY_KEY = ["session"] as const;

export function bootstrapQueryOptions(source: SessionSource) {
  return queryOptions({
    queryKey: SESSION_QUERY_KEY,
    queryFn: () => source.getSession(),
    // Infinity: cookie expiry is detected by the next RPC returning
    // Unauthenticated; login/logout mutations invalidate SESSION_QUERY_KEY.
    staleTime: Infinity,
  });
}
