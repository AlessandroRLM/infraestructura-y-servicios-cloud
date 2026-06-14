import { useInfiniteQuery } from "@connectrpc/connect-query";
import { CatalogService } from "@/gen/catalog/v1/catalog_pb";
import { CATALOG_PAGE_SIZE } from "../constants";

/**
 * Cursor-paginated list of Programs with optional search filter.
 *
 * Calls connect-query's useInfiniteQuery directly with the concrete listPrograms
 * descriptor — at a concrete call site the request type is known, so pageParamKey
 * and the input type check exactly with no casts. Splits the initial-load error
 * (no data yet → inline error + retry) from a fetchNextPage error (existing pages
 * remain visible).
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
