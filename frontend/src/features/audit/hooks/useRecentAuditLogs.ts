import { useInfiniteQuery } from "@connectrpc/connect-query";
import { AuditLogsService } from "@/gen/audit_logs/v1/audit_logs_pb";

const DEFAULT_PAGE_SIZE = 20;

interface UseRecentAuditLogsParams {
  actorId?: string;
  createdFrom?: string;
  createdTo?: string;
  pageSize?: number;
}

/**
 * Cursor-paginated global audit log feed. `isError` reflects only the initial
 * load; a failed "load more" surfaces via `isFetchNextPageError` while the
 * already-loaded pages stay visible.
 *
 * @param params - Optional filters: actorId, createdFrom, createdTo (RFC3339), pageSize.
 */
export function useRecentAuditLogs({
  actorId = "",
  createdFrom = "",
  createdTo = "",
  pageSize = DEFAULT_PAGE_SIZE,
}: UseRecentAuditLogsParams = {}) {
  const result = useInfiniteQuery(
    AuditLogsService.method.listRecentAuditLogs,
    { actorId, createdFrom, createdTo, pageSize, pageToken: "" },
    {
      pageParamKey: "pageToken",
      getNextPageParam: (lastPage) => lastPage.nextPageToken || undefined,
    },
  );

  const logs = result.data?.pages.flatMap((page) => page.logs) ?? [];
  const isFetchNextPageError = result.isFetchNextPageError;
  const isInitialLoadError = result.isError && !isFetchNextPageError;

  return {
    logs,
    isLoading: result.isLoading,
    isError: isInitialLoadError,
    refetch: result.refetch,
    fetchNextPage: result.fetchNextPage,
    hasNextPage: result.hasNextPage,
    isFetchingNextPage: result.isFetchingNextPage,
    isFetchNextPageError,
  };
}
