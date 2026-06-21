import { useInfiniteQuery } from "@connectrpc/connect-query";
import { CatalogService } from "@/gen/catalog/v1/catalog_pb";
import { DEFAULT_PAGE_SIZE } from "../pagination";

/**
 * Cursor-paginated list of Sections with optional courseId / academicPeriodId filters.
 * Mirrors the useCourses pattern — first-page error surfaces via isError; subsequent
 * page errors surface via isFetchNextPageError while prior pages stay visible.
 */
export function useSections(
  filters: { courseId?: string; academicPeriodId?: string } = {},
  pageSize: number = DEFAULT_PAGE_SIZE,
) {
  const result = useInfiniteQuery(
    CatalogService.method.listSections,
    {
      courseId: filters.courseId || undefined,
      academicPeriodId: filters.academicPeriodId || undefined,
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
