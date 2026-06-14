import { useCursorList } from "@/core/pagination";
import { IamService } from "@/gen/iam/v1/iam_pb";
import { USERS_PAGE_SIZE } from "../constants";

export function useUsersList(query: string) {
  const result = useCursorList(IamService.method.listUsers, {
    extract: (r) => ({ items: r.users, nextPageToken: r.nextPageToken }),
    baseInput: { query },
    pageSize: USERS_PAGE_SIZE,
  });

  // Separate the initial-load error (no data yet) from a fetchNextPage error
  // (existing pages remain). The initial load error renders inline + retry;
  // the next-page error fires a toast while keeping the current rows visible.
  const isFetchNextPageError = result.isFetchNextPageError;
  const isInitialLoadError = result.isError && !isFetchNextPageError;

  return {
    users: result.items,
    isLoading: result.isLoading,
    isError: isInitialLoadError,
    refetch: result.refetch,
    fetchNextPage: result.fetchNextPage,
    hasNextPage: result.hasNextPage,
    isFetchingNextPage: result.isFetchingNextPage,
    isFetchNextPageError,
  };
}
