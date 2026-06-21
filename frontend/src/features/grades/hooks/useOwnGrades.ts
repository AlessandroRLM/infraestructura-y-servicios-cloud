import { useInfiniteQuery } from "@connectrpc/connect-query";
import type {
  FetchNextPageOptions,
  InfiniteQueryObserverResult,
} from "@tanstack/react-query";
import { DEFAULT_PAGE_SIZE } from "@/core/pagination";
import { GradesService } from "@/gen/grades/v1/grades_pb";
import type { GradeSectionGroup } from "../groupBySection";
import { groupBySection } from "../groupBySection";

/** Infinite-query result for the student's own grades, pre-grouped by section. */
export interface UseOwnGradesResult {
  /** All loaded groups, accumulated across pages. */
  groups: GradeSectionGroup[];
  isLoading: boolean;
  /** True when the initial load failed (not a subsequent page failure). */
  isError: boolean;
  refetch: () => void;
  fetchNextPage: (
    options?: FetchNextPageOptions,
  ) => Promise<InfiniteQueryObserverResult>;
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  /** True when a "load more" page fetch failed (initial pages remain visible). */
  isFetchNextPageError: boolean;
}

/**
 * Infinite-cursor query for the authenticated student's own grades.
 * Filters are server-side: changing `academicPeriodId` or `programId`
 * remounts the query and resets to page 1. `pageSize` is part of the
 * connect-query key (method + input), so changing it also resets correctly.
 *
 * @param academicPeriodId - UUID string; empty string means no period filter.
 * @param programId - UUID string; empty string means no program filter.
 * @param pageSize - Maximum grades per page; clamped server-side to [20, 200].
 */
export function useOwnGrades(
  academicPeriodId: string,
  programId: string,
  pageSize: number = DEFAULT_PAGE_SIZE,
): UseOwnGradesResult {
  const result = useInfiniteQuery(
    GradesService.method.listOwnGrades,
    {
      pageSize,
      pageToken: "",
      academicPeriodId: academicPeriodId || undefined,
      programId: programId || undefined,
    },
    {
      pageParamKey: "pageToken",
      getNextPageParam: (lastPage) => lastPage.nextPageToken || undefined,
    },
  );

  const groups = groupBySection(
    result.data?.pages.flatMap((page) => page.grades) ?? [],
  );

  const isFetchNextPageError = result.isFetchNextPageError;
  const isInitialLoadError = result.isError && !isFetchNextPageError;

  return {
    groups,
    isLoading: result.isLoading,
    isError: isInitialLoadError,
    refetch: result.refetch,
    fetchNextPage: result.fetchNextPage,
    hasNextPage: result.hasNextPage ?? false,
    isFetchingNextPage: result.isFetchingNextPage,
    isFetchNextPageError,
  };
}

// Re-export the method reference for tests that need to mock at the service level.
export { GradesService };
