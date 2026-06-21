import { useInfiniteQuery } from "@connectrpc/connect-query";
import { DEFAULT_PAGE_SIZE } from "@/core/pagination";
import { EnrollmentService } from "@/gen/enrollment/v1/enrollment_pb";

/**
 * Cursor-paginated list of the authenticated student's own enrollments.
 * No filters — students see all their enrollments unconditionally.
 *
 * `isError` reflects only the initial load; a failed "load more" surfaces via
 * `isFetchNextPageError` while the already-loaded pages stay visible.
 */
export function useOwnEnrollments(pageSize: number = DEFAULT_PAGE_SIZE) {
  const result = useInfiniteQuery(
    EnrollmentService.method.listOwnEnrollments,
    {
      pageSize,
      pageToken: "",
    },
    {
      pageParamKey: "pageToken",
      getNextPageParam: (lastPage) => lastPage.nextPageToken || undefined,
    },
  );

  const enrollments =
    result.data?.pages.flatMap((page) => page.enrollments) ?? [];
  const isFetchNextPageError = result.isFetchNextPageError;
  const isInitialLoadError = result.isError && !isFetchNextPageError;

  return {
    enrollments,
    isLoading: result.isLoading,
    isError: isInitialLoadError,
    refetch: result.refetch,
    fetchNextPage: result.fetchNextPage,
    hasNextPage: result.hasNextPage,
    isFetchingNextPage: result.isFetchingNextPage,
    isFetchNextPageError,
  };
}
