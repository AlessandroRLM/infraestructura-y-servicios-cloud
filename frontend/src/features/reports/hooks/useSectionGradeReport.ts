import { useQuery } from "@connectrpc/connect-query";
import type {
  GetSectionGradeReportResponse,
  StudentGradeRow,
} from "@/gen/reports/v1/reports_pb";
import { ReportsService } from "@/gen/reports/v1/reports_pb";

/**
 * Typed result returned by useSectionGradeReport.
 * Never exposes the raw TanStack shape — callers interact with this contract only.
 */
export interface UseSectionGradeReportResult {
  /** The raw response when data is available; null when idle, loading, or errored. */
  data: GetSectionGradeReportResponse | null;
  rows: StudentGradeRow[];
  generatedAt: string;
  truncated: boolean;
  sectionId: string;
  isLoading: boolean;
  isFetching: boolean;
  isError: boolean;
  error: Error | null;
  refetch: () => void;
}

/**
 * Unary query for the section grade report.
 *
 * The query is enabled only when:
 *   1. `sectionId` is non-empty (filter is set), AND
 *   2. `isSectionGradeTabActive` is true (active tab guard per RF-2.7).
 *
 * Returns a typed result object — never the raw TanStack query shape.
 */
export function useSectionGradeReport(
  sectionId: string,
  isSectionGradeTabActive: boolean,
): UseSectionGradeReportResult {
  const enabled = !!sectionId && isSectionGradeTabActive;

  const result = useQuery(
    ReportsService.method.getSectionGradeReport,
    { sectionId },
    { enabled },
  );

  return {
    data: result.data ?? null,
    rows: result.data?.rows ?? [],
    generatedAt: result.data?.generatedAt ?? "",
    truncated: result.data?.truncated ?? false,
    sectionId: result.data?.sectionId ?? sectionId,
    isLoading: result.isLoading,
    isFetching: result.isFetching,
    isError: result.isError,
    error: result.error as Error | null,
    refetch: result.refetch,
  };
}

// Re-export the service reference for tests that stub at the service level.
export { ReportsService };
