import { useInfiniteQuery } from "@connectrpc/connect-query";
import { CatalogService } from "@/gen/catalog/v1/catalog_pb";
import { CATALOG_PAGE_SIZE } from "../constants";

/**
 * Cursor-paginated list of Programs with optional search filter. `isError`
 * reflects only the initial load; a failed "load more" surfaces via
 * `isFetchNextPageError` while the loaded pages stay visible.
 */
export function usePrograms(query = "", pageSize: number = CATALOG_PAGE_SIZE) {
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
