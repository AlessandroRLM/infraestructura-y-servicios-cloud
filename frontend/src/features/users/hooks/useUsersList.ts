import { useInfiniteQuery } from "@connectrpc/connect-query";
import { DEFAULT_PAGE_SIZE } from "@/core/pagination";
import { IamService } from "@/gen/iam/v1/iam_pb";

/**
 * Cursor-paginated list of users with optional search filter. `isError`
 * reflects only the initial load; a failed "load more" surfaces via
 * `isFetchNextPageError` while the loaded pages stay visible.
 */
export function useUsersList(
  query: string,
  pageSize: number = DEFAULT_PAGE_SIZE,
) {
  const result = useInfiniteQuery(
    IamService.method.listUsers,
    { query, pageSize, pageToken: "" },
    {
      pageParamKey: "pageToken",
      getNextPageParam: (lastPage) => lastPage.nextPageToken || undefined,
    },
  );

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
