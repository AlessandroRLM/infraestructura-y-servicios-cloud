import type { GetSectionOccupancyReportResponse } from "@/gen/reports/v1/reports_pb";
import type { ReportPdfModel } from "./model";

/** Truncation cap for SectionOccupancy as specified in RF-4.6. */
const SECTION_OCCUPANCY_TRUNCATION_CAP = 1000;

/**
 * Maps a GetSectionOccupancyReportResponse to the normalized ReportPdfModel.
 *
 * Column layout:
 *   - Curso (course name)       — text, left
 *   - Sección (section id)      — text, left
 *   - Cupo (capacity)           — numeric, right
 *   - Inscritos (activeSeatCount) — numeric, right
 *   - % Ocupación (fillPercentage) — numeric, right
 *
 * Column widths sum to 100 (%).
 * When truncated is true, sets truncatedTo = 1000 and bakes notice into footer.
 *
 * The mapper is pure — no hooks, no side effects, safe to unit-test directly.
 */
export function toSectionOccupancyReportModel(
  response: GetSectionOccupancyReportResponse,
  periodLabel: string,
): ReportPdfModel {
  const { rows, generatedAt, truncated, academicPeriodName } = response;

  // Use academicPeriodName from the response when available; fall back to the
  // caller-supplied label (from the picker's selectedLabel or URL param).
  const filterLabel = academicPeriodName || periodLabel;

  const columns: ReportPdfModel["columns"] = [
    { key: "curso", label: "Curso", width: 30, align: "left" },
    { key: "seccion", label: "Sección", width: 20, align: "left" },
    { key: "cupo", label: "Cupo", width: 15, align: "right" },
    { key: "inscritos", label: "Inscritos", width: 15, align: "right" },
    { key: "ocupacion", label: "% Ocupación", width: 20, align: "right" },
  ];

  const mappedRows: string[][] = rows.map((row) => [
    row.courseName || "—",
    row.sectionId.slice(0, 8) + "…",
    String(row.capacity),
    String(row.activeSeatCount),
    `${row.fillPercentage}%`,
  ]);

  const truncatedTo = truncated ? SECTION_OCCUPANCY_TRUNCATION_CAP : undefined;
  const footer = truncated
    ? `Documento truncado a ${SECTION_OCCUPANCY_TRUNCATION_CAP} filas`
    : `Reporte generado el ${generatedAt}`;

  return {
    title: "Ocupación por Período",
    appliedFilter: filterLabel,
    generatedAt,
    truncatedTo,
    columns,
    rows: mappedRows,
    footer,
  };
}

export { SECTION_OCCUPANCY_TRUNCATION_CAP };
