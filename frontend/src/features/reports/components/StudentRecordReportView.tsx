import { usePdfBlob } from "../hooks/usePdfBlob";
import { useStudentRecordReport } from "../hooks/useStudentRecordReport";
import { toStudentRecordReportModel } from "../pdf/toStudentRecordReportModel";
import { ReportPdfPreview } from "./ReportPdfPreview";
import { ReportStateBoundary } from "./ReportStateBoundary";
import { StudentPicker } from "./StudentPicker";

interface StudentRecordReportViewProps {
  /** The currently selected student ID (from URL search param). */
  studentId: string;
  /** Whether the student-record tab is the active tab (controls query enabled flag). */
  isActive: boolean;
  /** Called when the user selects a different student in the picker. */
  onStudentChange: (studentId: string) => void;
}

/**
 * StudentRecord tab panel (presentational side).
 *
 * Props come from the router-connected parent (ReportsPage).
 * Contains NO Route.useSearch() calls — props-driven for testability.
 *
 * Wiring:
 *   StudentPicker → onStudentChange → studentId in URL param
 *   → useStudentRecordReport (enabled when studentId set + isActive)
 *   → toStudentRecordReportModel (pure mapper)
 *   → usePdfBlob (renders PDF, owns blob lifecycle)
 *   → ReportPdfPreview (iframe + download)
 *
 * Hard rules:
 *   - NO useMemo/useCallback — the React Compiler handles memoization.
 */
export function StudentRecordReportView({
  studentId,
  isActive,
  onStudentChange,
}: StudentRecordReportViewProps) {
  const report = useStudentRecordReport(studentId, isActive);

  const pdfModel =
    report.data !== null ? toStudentRecordReportModel(report.data) : null;

  const { url, isRendering, error: blobError } = usePdfBlob(pdfModel);

  const filterSet = !!studentId;
  const isEmpty =
    !report.isLoading &&
    !report.isFetching &&
    !report.isError &&
    report.data !== null &&
    report.rows.length === 0;

  const isError = report.isError || blobError !== null;
  const error = report.error ?? blobError;

  return (
    <div className="flex flex-col gap-4">
      <div className="max-w-sm">
        <StudentPicker value={studentId} onChange={onStudentChange} />
      </div>

      <ReportStateBoundary
        filterSet={filterSet}
        isFetching={report.isFetching}
        isRendering={isRendering}
        isError={isError}
        error={error}
        isEmpty={isEmpty}
        {...(report.truncated
          ? { truncated: true as const, truncatedTo: 1000 }
          : { truncated: false as const })}
        onRetry={report.refetch}
      >
        <ReportPdfPreview
          url={url}
          isRendering={isRendering}
          fileName="expediente-alumno.pdf"
          generatedAt={report.generatedAt || undefined}
        />
      </ReportStateBoundary>
    </div>
  );
}
