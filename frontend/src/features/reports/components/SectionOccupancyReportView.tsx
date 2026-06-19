import { usePdfBlob } from "../hooks/usePdfBlob";
import { useSectionOccupancyReport } from "../hooks/useSectionOccupancyReport";
import { toSectionOccupancyReportModel } from "../pdf/toSectionOccupancyReportModel";
import { AcademicPeriodPicker } from "./AcademicPeriodPicker";
import { ReportPdfPreview } from "./ReportPdfPreview";
import { ReportStateBoundary } from "./ReportStateBoundary";

interface SectionOccupancyReportViewProps {
  /** The currently selected academic period ID (from URL search param). */
  periodId: string;
  /** Whether the occupancy tab is the active tab (controls query enabled flag). */
  isActive: boolean;
  /** Called when the user selects a different period in the picker. */
  onPeriodChange: (periodId: string) => void;
}

/**
 * SectionOccupancy tab panel (presentational side).
 *
 * Props come from the router-connected parent (ReportsPage).
 * Contains NO Route.useSearch() calls — props-driven for testability.
 *
 * Wiring:
 *   AcademicPeriodPicker → onPeriodChange → periodId in URL param
 *   → useSectionOccupancyReport (enabled when periodId set + isActive)
 *   → toSectionOccupancyReportModel (pure mapper)
 *   → usePdfBlob (renders PDF, owns blob lifecycle)
 *   → ReportPdfPreview (iframe + download)
 *
 * Hard rules:
 *   - NO useMemo/useCallback — the React Compiler handles memoization.
 */
export function SectionOccupancyReportView({
  periodId,
  isActive,
  onPeriodChange,
}: SectionOccupancyReportViewProps) {
  const report = useSectionOccupancyReport(periodId, isActive);

  const pdfModel =
    report.data !== null
      ? toSectionOccupancyReportModel(report.data, periodId)
      : null;

  const { url, isRendering, error: blobError } = usePdfBlob(pdfModel);

  const filterSet = !!periodId;
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
        <AcademicPeriodPicker value={periodId} onChange={onPeriodChange} />
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
          fileName="ocupacion-por-periodo.pdf"
          generatedAt={report.generatedAt || undefined}
        />
      </ReportStateBoundary>
    </div>
  );
}
