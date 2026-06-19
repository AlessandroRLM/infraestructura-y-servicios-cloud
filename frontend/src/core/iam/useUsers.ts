import { useInfiniteQuery } from "@connectrpc/connect-query";
import { IamService } from "@/gen/iam/v1/iam_pb";

/**
 * Cursor-paginated hook for listing users with optional text search (email / display_name).
 * Lives in core/iam/ so features/reports (and others) can consume it without
 * deep-importing from features/users — respecting the feature isolation convention.
 *
 * Backed by IamService.listUsers (requires users.manage at the service layer).
 */
export function useUsers(query: string, pageSize = 50) {
  const result = useInfiniteQuery(
    IamService.method.listUsers,
    { query, pageSize, pageToken: "" },
    {
      pageParamKey: "pageToken",
      getNextPageParam: (lastPage) => lastPage.nextPageToken || undefined,
    },
  );

  const users = result.data?.pages.flatMap((page) => page.users) ?? [];

  return {
    users,
    isLoading: result.isLoading,
    isError: result.isError,
    hasNextPage: result.hasNextPage,
    isFetchingNextPage: result.isFetchingNextPage,
    fetchNextPage: result.fetchNextPage,
    refetch: result.refetch,
  };
}
