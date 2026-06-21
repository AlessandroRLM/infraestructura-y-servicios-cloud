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

/** Parameters accepted by useOwnSections. */
export interface UseOwnSectionsParams {
  /** Maximum rows per page; defaults to 50. */
  pageSize?: number;
  /**
   * Optional server-side search string.
   * ILIKE-matched against course code and name.
   * Changing this value resets pagination to page 1.
   */
  query?: string;
  /**
   * When false, the query is skipped entirely (no RPC fired).
   * Defaults to true. Pass false when the section is already known
   * from router navigation state to avoid a redundant ListOwnSections call.
   */
  enabled?: boolean;
}

/**
 * Cursor-paginated list of teaching sections for the authenticated caller.
 *
 * Role-agnostic: the backend discriminator handles the distinction:
 *   - Teachers (section.view_teaching, no catalog.manage) → their own sections only.
 *   - Admins (catalog.manage) → all sections.
 *
 * Mirrors the ProgramsTable usePrograms pattern (infinite-query + pagination).
 * Changing `query` resets pagination to page 1 (new query key).
 *
 * @param params - Optional pageSize and query parameters.
 */
export function useOwnSections(
  params: UseOwnSectionsParams = {},
): UseOwnSectionsResult {
  const { pageSize = SECTIONS_PAGE_SIZE, query = "", enabled = true } = params;
  const result = useInfiniteQuery(
    CatalogService.method.listOwnSections,
    { pageSize, pageToken: "", query },
    {
      pageParamKey: "pageToken",
      getNextPageParam: (lastPage) => lastPage.nextPageToken || undefined,
      enabled,
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
