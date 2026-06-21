import { useInfiniteQuery } from "@connectrpc/connect-query";
import { CatalogService } from "@/gen/catalog/v1/catalog_pb";
import { DEFAULT_PAGE_SIZE } from "../pagination";

/**
 * Cursor-paginated list of Programs with optional search filter.
 * Mirrors the useCourses pattern — first-page error surfaces via isError;
 * subsequent page errors surface via isFetchNextPageError while prior pages stay visible.
 */
export function usePrograms(query = "", pageSize: number = DEFAULT_PAGE_SIZE) {
  const result = useInfiniteQuery(
    CatalogService.method.listPrograms,
    { query, pageSize, pageToken: "" },
    {
      pageParamKey: "pageToken",
      getNextPageParam: (lastPage) => lastPage.nextPageToken || undefined,
    },
  );

  const programs = result.data?.pages.flatMap((page) => page.programs) ?? [];
  const isFetchNextPageError = result.isFetchNextPageError;
  const isInitialLoadError = result.isError && !isFetchNextPageError;

  return {
    programs,
    isLoading: result.isLoading,
    isError: isInitialLoadError,
    refetch: result.refetch,
    fetchNextPage: result.fetchNextPage,
    hasNextPage: result.hasNextPage,
    isFetchingNextPage: result.isFetchingNextPage,
    isFetchNextPageError,
  };
}
