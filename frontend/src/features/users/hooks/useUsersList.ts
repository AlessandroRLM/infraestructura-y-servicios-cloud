import { useInfiniteQuery } from "@connectrpc/connect-query";
import { IamService } from "@/gen/iam/v1/iam_pb";
import { USERS_PAGE_SIZE } from "../constants";

/**
 * Cursor-paginated list of users with optional search filter.
 *
 * Calls connect-query's useInfiniteQuery directly with the concrete listUsers
 * descriptor — at a concrete call site the request type is known, so pageParamKey
 * and the input type check exactly with no casts.
 */
export function useUsersList(
  query: string,
  pageSize: number = USERS_PAGE_SIZE,
) {
  const result = useInfiniteQuery(
    IamService.method.listUsers,
    { query, pageSize, pageToken: "" },
    {
      pageParamKey: "pageToken",
      getNextPageParam: (lastPage) => lastPage.nextPageToken || undefined,
    },
  );

  // Separate the initial-load error (no data yet) from a fetchNextPage error
  // (existing pages remain). The initial load error renders inline + retry;
  // the next-page error fires a toast while keeping the current rows visible.
  const users = result.data?.pages.flatMap((page) => page.users) ?? [];
  const isFetchNextPageError = result.isFetchNextPageError;
  const isInitialLoadError = result.isError && !isFetchNextPageError;

  return {
    users,
    isLoading: result.isLoading,
    isError: isInitialLoadError,
    refetch: result.refetch,
    fetchNextPage: result.fetchNextPage,
    hasNextPage: result.hasNextPage,
    isFetchingNextPage: result.isFetchingNextPage,
    isFetchNextPageError,
  };
}
