import { useQuery } from "@connectrpc/connect-query";
import type {
  GetProgramSummaryReportResponse,
  ProgramEnrollmentRow,
} from "@/gen/reports/v1/reports_pb";
import { ReportsService } from "@/gen/reports/v1/reports_pb";

/**
 * Typed result returned by useProgramSummaryReport.
 */
export interface UseProgramSummaryReportResult {
  data: GetProgramSummaryReportResponse | null;
  rows: ProgramEnrollmentRow[];
  generatedAt: string;
  truncated: boolean;
  programId: string;
  programName: string;
  year: number;
  isLoading: boolean;
  isFetching: boolean;
  isError: boolean;
  error: Error | null;
  refetch: () => void;
}

/**
 * Unary query for the program summary report.
 *
 * Enabled only when:
 *   1. `programId` is non-empty AND `year` is defined (both filters set), AND
 *   2. `isProgramSummaryTabActive` is true (active tab guard per RF-2.7).
 */
export function useProgramSummaryReport(
  programId: string,
  year: number | undefined,
  isProgramSummaryTabActive: boolean,
): UseProgramSummaryReportResult {
  const enabled = !!programId && !!year && isProgramSummaryTabActive;

  const result = useQuery(
    ReportsService.method.getProgramSummaryReport,
    { programId, year: year ?? 0 },
    { enabled },
  );

  return {
    data: result.data ?? null,
    rows: result.data?.rows ?? [],
    generatedAt: result.data?.generatedAt ?? "",
    truncated: result.data?.truncated ?? false,
    programId: result.data?.programId ?? programId,
    programName: result.data?.programName ?? "",
    year: result.data?.year ?? year ?? 0,
    isLoading: result.isLoading,
    isFetching: result.isFetching,
    isError: result.isError,
    error: result.error as Error | null,
    refetch: result.refetch,
  };
}

export { ReportsService };
