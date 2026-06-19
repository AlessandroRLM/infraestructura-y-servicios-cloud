import { useInfiniteQuery } from "@connectrpc/connect-query";
import type { TeachingSection } from "@/gen/catalog/v1/catalog_pb";
import { CatalogService } from "@/gen/catalog/v1/catalog_pb";

const SECTIONS_PAGE_SIZE = 50;

/** Result shape returned by useOwnSections. */
export interface UseOwnSectionsResult {
  /** Flat list of TeachingSection rows across all loaded pages. */
  sections: TeachingSection[];
  /** True while the initial page is loading. */
  isLoading: boolean;
  /** True when the initial load failed. */
  isError: boolean;
  /** Re-triggers the initial query. */
  refetch: () => void;
  /** Loads the next page if one exists. */
  fetchNextPage: () => Promise<void>;
  /** True when another page is available. */
  hasNextPage: boolean;
  /** True while a "load more" fetch is in flight. */
  isFetchingNextPage: boolean;
  /** True when a "load more" page failed without affecting the already-loaded pages. */
  isFetchNextPageError: boolean;
}

/**
 * Cursor-paginated list of teaching sections for the authenticated caller.
 *
 * Role-agnostic: the backend discriminator handles the distinction:
 *   - Teachers (section.view_teaching, no catalog.manage) → their own sections only.
 *   - Admins (catalog.manage) → all sections.
 *
 * Mirrors the ProgramsTable usePrograms pattern (infinite-query + pagination).
 *
 * @param pageSize - Maximum rows per page; defaults to 50.
 */
export function useOwnSections(
  pageSize: number = SECTIONS_PAGE_SIZE,
): UseOwnSectionsResult {
  const result = useInfiniteQuery(
    CatalogService.method.listOwnSections,
    { pageSize, pageToken: "" },
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
    fetchNextPage: async () => {
      await result.fetchNextPage({ throwOnError: true });
    },
    hasNextPage: result.hasNextPage,
    isFetchingNextPage: result.isFetchingNextPage,
    isFetchNextPageError,
  };
}
