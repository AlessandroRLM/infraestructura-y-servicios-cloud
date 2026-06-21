import { useInfiniteQuery } from "@connectrpc/connect-query";
import { DEFAULT_PAGE_SIZE } from "@/core/pagination";
import { EnrollmentService } from "@/gen/enrollment/v1/enrollment_pb";

interface EnrollmentFilters {
  q?: string;
  year?: number;
  status?: string;
  pageSize?: number;
}

/**
 * Cursor-paginated list of enrollments for the admin view, with optional
 * free-text search and status / year filters.
 *
 * `isError` reflects only the initial load; a failed "load more" surfaces via
 * `isFetchNextPageError` while the already-loaded pages stay visible.
 */
export function useEnrollments({
  q = "",
  year,
  status,
  pageSize = DEFAULT_PAGE_SIZE,
}: EnrollmentFilters = {}) {
  const result = useInfiniteQuery(
    EnrollmentService.method.listEnrollments,
    {
      query: q,
      year: year ?? 0,
      status: status ?? "",
      studentId: "",
      programId: "",
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
