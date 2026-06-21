import { useInfiniteQuery } from "@connectrpc/connect-query";
import { DEFAULT_PAGE_SIZE } from "@/core/pagination";
import { SectionEnrollmentService } from "@/gen/section_enrollment/v1/section_enrollment_pb";

/**
 * Cursor-paginated list of sections the authenticated student is eligible to enroll in.
 * No filters — backend derives student identity and eligibility from session context.
 *
 * `isError` reflects only the initial load; a failed "load more" surfaces via
 * `isFetchNextPageError` while already-loaded pages stay visible.
 */
export function useEnrollableSections(pageSize: number = DEFAULT_PAGE_SIZE) {
  const result = useInfiniteQuery(
    SectionEnrollmentService.method.listEnrollableSections,
    {
      pageSize,
      pageToken: "",
    },
    {
      pageParamKey: "pageToken",
      getNextPageParam: (lastPage) => lastPage.nextPageToken || undefined,
    },
  );

  const sections = result.data?.pages.flatMap((page) => page.sections) ?? [];
  const isFetchNextPageError = result.isFetchNextPageError;
  const isInitialLoadError = result.isError && !isFetchNextPageError;

  return {
    sections,
    isLoading: result.isLoading,
    isError: isInitialLoadError,
    refetch: result.refetch,
    fetchNextPage: result.fetchNextPage,
    hasNextPage: result.hasNextPage,
    isFetchingNextPage: result.isFetchingNextPage,
    isFetchNextPageError,
  };
}
