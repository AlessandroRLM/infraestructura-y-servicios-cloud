import { useTransport } from "@connectrpc/connect-query";
import type {
  FetchNextPageOptions,
  InfiniteQueryObserverResult,
} from "@tanstack/react-query";
import { useInfiniteQuery } from "@tanstack/react-query";
import type { OwnGrade } from "@/gen/grades/v1/grades_pb";
import { GradesService } from "@/gen/grades/v1/grades_pb";
import { ownGradesQueryKey } from "../api/queries";
import { createRpcOwnGradesSource } from "../api/rpc";
import type { GradeSectionGroup } from "../groupBySection";
import { groupBySection } from "../groupBySection";

/** Default page size used when the URL param is absent or invalid. */
export const OWN_GRADES_DEFAULT_PAGE_SIZE = 20;

/** Infinite-query result for the student's own grades, pre-grouped by section. */
export interface UseOwnGradesResult {
  /** All loaded groups, accumulated across pages. */
  groups: GradeSectionGroup[];
  /** Flat raw grade rows across all loaded pages; used for deriving filter options. */
  rawGrades: OwnGrade[];
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
 * remounts the query and resets to page 1.
 *
 * @param academicPeriodId - UUID string; empty string means no period filter.
 * @param programId - UUID string; empty string means no program filter.
 * @param pageSize - Maximum grades per page; clamped server-side to [20, 200].
 */
export function useOwnGrades(
  academicPeriodId: string,
  programId: string,
  pageSize: number = OWN_GRADES_DEFAULT_PAGE_SIZE,
): UseOwnGradesResult {
  const transport = useTransport();
  const source = createRpcOwnGradesSource(transport);

  const result = useInfiniteQuery({
    queryKey: ownGradesQueryKey(academicPeriodId, programId),
    queryFn: async ({ pageParam }) => {
      return source.listOwnGrades({
        pageSize,
        pageToken: pageParam,
        academicPeriodId: academicPeriodId || undefined,
        programId: programId || undefined,
      });
    },
    initialPageParam: "",
    getNextPageParam: (lastPage) => lastPage.nextPageToken || undefined,
  });

  const rawGrades: OwnGrade[] =
    result.data?.pages.flatMap((page) => page.grades) ?? [];
  const groups = groupBySection(rawGrades);

  const isFetchNextPageError = result.isFetchNextPageError;
  const isInitialLoadError = result.isError && !isFetchNextPageError;

  return {
    groups,
    rawGrades,
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
