import { useInfiniteQuery } from "@connectrpc/connect-query";
import { SectionEnrollmentService } from "@/gen/section_enrollment/v1/section_enrollment_pb";

const SECTION_ENROLLMENTS_PAGE_SIZE = 20;

interface SectionEnrollmentsFilters {
  sectionId: string;
  status?: string;
  pageSize?: number;
}

/**
 * Cursor-paginated list of section enrollments for a given section (admin view).
 * isError reflects only the initial load; a failed "load more" surfaces via
 * isFetchNextPageError while the already-loaded pages stay visible.
 */
export function useSectionEnrollments({
  sectionId,
  status,
  pageSize = SECTION_ENROLLMENTS_PAGE_SIZE,
}: SectionEnrollmentsFilters) {
  const result = useInfiniteQuery(
    SectionEnrollmentService.method.listSectionEnrollments,
    {
      sectionId,
      enrollmentId: "",
      status: status ?? "",
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
