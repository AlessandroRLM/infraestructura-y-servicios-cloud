import { useInfiniteQuery } from "@connectrpc/connect-query";
import { DEFAULT_PAGE_SIZE } from "@/core/pagination";
import { SectionEnrollmentService } from "@/gen/section_enrollment/v1/section_enrollment_pb";

/**
 * Cursor-paginated list of the authenticated student's own section enrollments.
 * No filters — student identity is derived from session context by the backend.
 *
 * `isError` reflects only the initial load; a failed "load more" surfaces via
 * `isFetchNextPageError` while the already-loaded pages stay visible.
 */
export function useOwnSectionEnrollments(pageSize: number = DEFAULT_PAGE_SIZE) {
  const result = useInfiniteQuery(
    SectionEnrollmentService.method.listOwnSectionEnrollments,
    {
      pageSize,
      pageToken: "",
    },
    {
      pageParamKey: "pageToken",
      getNextPageParam: (lastPage) => lastPage.nextPageToken || undefined,
    },
  );

  const sectionEnrollments =
    result.data?.pages.flatMap((page) => page.sectionEnrollments) ?? [];
  const isFetchNextPageError = result.isFetchNextPageError;
  const isInitialLoadError = result.isError && !isFetchNextPageError;

  return {
    sectionEnrollments,
    isLoading: result.isLoading,
    isError: isInitialLoadError,
    refetch: result.refetch,
    fetchNextPage: result.fetchNextPage,
    hasNextPage: result.hasNextPage,
    isFetchingNextPage: result.isFetchingNextPage,
    isFetchNextPageError,
  };
}
