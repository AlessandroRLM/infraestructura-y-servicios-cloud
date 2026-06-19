import { useQuery } from "@connectrpc/connect-query";
import type {
  AcademicRecordRow,
  GetStudentRecordReportResponse,
} from "@/gen/reports/v1/reports_pb";
import { ReportsService } from "@/gen/reports/v1/reports_pb";

/**
 * Typed result returned by useStudentRecordReport.
 */
export interface UseStudentRecordReportResult {
  data: GetStudentRecordReportResponse | null;
  rows: AcademicRecordRow[];
  generatedAt: string;
  truncated: boolean;
  studentId: string;
  studentName: string;
  isLoading: boolean;
  isFetching: boolean;
  isError: boolean;
  error: Error | null;
  refetch: () => void;
}

/**
 * Unary query for the student record report (cross-period academic history).
 *
 * Enabled only when:
 *   1. `studentId` is non-empty (filter is set), AND
 *   2. `isStudentRecordTabActive` is true (active tab guard per RF-2.7).
 *
 * Truncation cap: 1000 rows (RF-4.6).
 */
export function useStudentRecordReport(
  studentId: string,
  isStudentRecordTabActive: boolean,
): UseStudentRecordReportResult {
  const enabled = !!studentId && isStudentRecordTabActive;

  const result = useQuery(
    ReportsService.method.getStudentRecordReport,
    { studentId },
    { enabled },
  );

  return {
    data: result.data ?? null,
    rows: result.data?.rows ?? [],
    generatedAt: result.data?.generatedAt ?? "",
    truncated: result.data?.truncated ?? false,
    studentId: result.data?.studentId ?? studentId,
    studentName: result.data?.studentName ?? "",
    isLoading: result.isLoading,
    isFetching: result.isFetching,
    isError: result.isError,
    error: result.error instanceof Error ? result.error : null,
    refetch: result.refetch,
  };
}

export { ReportsService };
