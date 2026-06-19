import { useQuery } from "@connectrpc/connect-query";
import type {
  GetSectionOccupancyReportResponse,
  SectionOccupancyRow,
} from "@/gen/reports/v1/reports_pb";
import { ReportsService } from "@/gen/reports/v1/reports_pb";

/**
 * Typed result returned by useSectionOccupancyReport.
 */
export interface UseSectionOccupancyReportResult {
  data: GetSectionOccupancyReportResponse | null;
  rows: SectionOccupancyRow[];
  generatedAt: string;
  truncated: boolean;
  academicPeriodId: string;
  academicPeriodName: string;
  isLoading: boolean;
  isFetching: boolean;
  isError: boolean;
  error: Error | null;
  refetch: () => void;
}

/**
 * Unary query for the section occupancy report.
 *
 * Enabled only when:
 *   1. `periodId` is non-empty (filter is set), AND
 *   2. `isOccupancyTabActive` is true (active tab guard per RF-2.7).
 */
export function useSectionOccupancyReport(
  periodId: string,
  isOccupancyTabActive: boolean,
): UseSectionOccupancyReportResult {
  const enabled = !!periodId && isOccupancyTabActive;

  const result = useQuery(
    ReportsService.method.getSectionOccupancyReport,
    { academicPeriodId: periodId },
    { enabled },
  );

  return {
    data: result.data ?? null,
    rows: result.data?.rows ?? [],
    generatedAt: result.data?.generatedAt ?? "",
    truncated: result.data?.truncated ?? false,
    academicPeriodId: result.data?.academicPeriodId ?? periodId,
    academicPeriodName: result.data?.academicPeriodName ?? "",
    isLoading: result.isLoading,
    isFetching: result.isFetching,
    isError: result.isError,
    error: result.error as Error | null,
    refetch: result.refetch,
  };
}

export { ReportsService };
