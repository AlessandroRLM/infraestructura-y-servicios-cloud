import { Code, ConnectError } from "@connectrpc/connect";
import { queryOptions } from "@tanstack/react-query";
import type { ProfileSource } from "./rpc";

export const OWN_PROFILE_QUERY_KEY = ["profile", "own"] as const;

function notFoundAwareRetry(count: number, err: unknown): boolean {
  if (err instanceof ConnectError && err.code === Code.NotFound) {
    return false;
  }
  return count < 3;
}

export function ownProfileQueryOptions(source: ProfileSource) {
  return queryOptions({
    queryKey: OWN_PROFILE_QUERY_KEY,
    queryFn: () => source.getOwnProfile(),
    staleTime: 30_000,
    retry: notFoundAwareRetry,
  });
}
