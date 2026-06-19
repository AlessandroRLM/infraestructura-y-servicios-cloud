import { usePdfBlob } from "../hooks/usePdfBlob";
import { useSectionGradeReport } from "../hooks/useSectionGradeReport";
import { toSectionGradeReportModel } from "../pdf/toSectionGradeReportModel";
import { ReportPdfPreview } from "./ReportPdfPreview";
import { ReportStateBoundary } from "./ReportStateBoundary";
import { SectionPicker } from "./SectionPicker";

interface SectionGradeReportViewProps {
  /** The currently selected section ID (from URL search param). */
  sectionId: string;
  /** Whether the section-grade tab is the active tab (controls query enabled flag). */
  isActive: boolean;
  /** Called when the user selects a different section in the picker. */
  onSectionChange: (sectionId: string) => void;
}

/**
 * SectionGrade tab panel (presentational side).
 *
 * Props come from the router-connected parent (ReportsPage / ReportsTabShell).
 * This component contains NO Route.useSearch() calls — it only receives what
 * it needs as props, which makes it testable without a router context.
 *
 * Wiring:
 *   SectionPicker → onSectionChange → sectionId in URL param (managed by parent)
 *   → useSectionGradeReport (enabled when sectionId set + isActive)
 *   → toSectionGradeReportModel (pure mapper, Compiler-memoized at render time)
 *   → usePdfBlob (renders PDF, owns blob lifecycle)
 *   → ReportPdfPreview (iframe + download)
 *
 * Hard rules:
 *   - NO useMemo/useCallback — the React Compiler handles memoization.
 *   - toSectionGradeReportModel runs at render time; TanStack Query data is
 *     reference-stable across renders so the Compiler can skip re-computation.
 *   - usePdfBlob receives null when no data is available → stays idle, no render.
 */
export function SectionGradeReportView({
  sectionId,
  isActive,
  onSectionChange,
}: SectionGradeReportViewProps) {
  const report = useSectionGradeReport(sectionId, isActive);

  // Derive the PDF model from the response data.
  // The React Compiler memoizes this — no useMemo wrapper needed.
  const pdfModel =
    report.data !== null
      ? toSectionGradeReportModel(report.data, sectionId)
      : null;

  const { url, isRendering, error: blobError } = usePdfBlob(pdfModel);

  const filterSet = !!sectionId;
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
      {/* Filter picker — always visible so the user can change the section */}
      <div className="max-w-sm">
        <SectionPicker value={sectionId} onChange={onSectionChange} />
      </div>

      <ReportStateBoundary
        filterSet={filterSet}
        isFetching={report.isFetching}
        isRendering={isRendering}
        isError={isError}
        error={error}
        isEmpty={isEmpty}
        {...(report.truncated
          ? { truncated: true as const, truncatedTo: 500 }
          : { truncated: false as const })}
        onRetry={report.refetch}
      >
        <ReportPdfPreview
          url={url}
          isRendering={isRendering}
          fileName="calificaciones-por-seccion.pdf"
          generatedAt={report.generatedAt || undefined}
        />
      </ReportStateBoundary>
    </div>
  );
}
