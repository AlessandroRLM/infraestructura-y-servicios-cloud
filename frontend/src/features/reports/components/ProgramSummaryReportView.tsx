import { usePdfBlob } from "../hooks/usePdfBlob";
import { useProgramSummaryReport } from "../hooks/useProgramSummaryReport";
import { toProgramSummaryReportModel } from "../pdf/toProgramSummaryReportModel";
import { ProgramYearPicker } from "./ProgramYearPicker";
import { ReportPdfPreview } from "./ReportPdfPreview";
import { ReportStateBoundary } from "./ReportStateBoundary";

interface ProgramSummaryReportViewProps {
  /** The currently selected program ID (from URL search param). */
  programId: string;
  /** The currently selected year (from URL search param). */
  year: number | undefined;
  /** Whether the program-summary tab is the active tab. */
  isActive: boolean;
  /** Called when the user selects a different program. */
  onProgramChange: (programId: string) => void;
  /** Called when the user changes the year. */
  onYearChange: (year: number | undefined) => void;
}

/**
 * ProgramSummary tab panel (presentational side).
 *
 * Props come from the router-connected parent (ReportsPage).
 * Contains NO Route.useSearch() calls — props-driven for testability.
 *
 * Wiring:
 *   ProgramYearPicker → onProgramChange/onYearChange → params in URL
 *   → useProgramSummaryReport (enabled when both set + isActive)
 *   → toProgramSummaryReportModel (pure mapper)
 *   → usePdfBlob (renders PDF, owns blob lifecycle)
 *   → ReportPdfPreview (iframe + download)
 *
 * Hard rules:
 *   - NO useMemo/useCallback — the React Compiler handles memoization.
 */
export function ProgramSummaryReportView({
  programId,
  year,
  isActive,
  onProgramChange,
  onYearChange,
}: ProgramSummaryReportViewProps) {
  const report = useProgramSummaryReport(programId, year, isActive);

  const pdfModel =
    report.data !== null
      ? toProgramSummaryReportModel(report.data, programId)
      : null;

  const { url, isRendering, error: blobError } = usePdfBlob(pdfModel);

  const filterSet = !!programId && !!year;
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
        <ProgramYearPicker
          programId={programId}
          year={year}
          onProgramChange={onProgramChange}
          onYearChange={onYearChange}
        />
      </div>

      <ReportStateBoundary
        filterSet={filterSet}
        isFetching={report.isFetching}
        isRendering={isRendering}
        isError={isError}
        error={error}
        isEmpty={isEmpty}
        {...(report.truncated
          ? { truncated: true as const, truncatedTo: 200 }
          : { truncated: false as const })}
        onRetry={report.refetch}
      >
        <ReportPdfPreview
          url={url}
          isRendering={isRendering}
          fileName="resumen-de-programa.pdf"
          generatedAt={report.generatedAt || undefined}
        />
      </ReportStateBoundary>
    </div>
  );
}
