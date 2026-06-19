import type { GetProgramSummaryReportResponse } from "@/gen/reports/v1/reports_pb";
import type { ReportPdfModel } from "./model";

/** Truncation cap for ProgramSummary as specified in RF-4.6. */
const PROGRAM_SUMMARY_TRUNCATION_CAP = 200;

/**
 * Maps a GetProgramSummaryReportResponse to the normalized ReportPdfModel.
 *
 * Column layout:
 *   - Programa (programName)          — text, left
 *   - Cupo (quotaCapacity)            — numeric, right
 *   - Inscritos (enrolledCount)       — numeric, right
 *   - Disponibles (availableSeats)    — numeric, right
 *
 * Column widths sum to 100 (%).
 * When truncated is true, sets truncatedTo = 200 and bakes notice into footer.
 *
 * The mapper is pure — no hooks, no side effects, safe to unit-test directly.
 */
export function toProgramSummaryReportModel(
  response: GetProgramSummaryReportResponse,
  programLabel: string,
): ReportPdfModel {
  const { rows, generatedAt, truncated, programName, year } = response;

  const filterLabel = programName
    ? `${programName} — ${year}`
    : programLabel || `Año ${year}`;

  const columns: ReportPdfModel["columns"] = [
    { key: "programa", label: "Programa", width: 40, align: "left" },
    { key: "cupo", label: "Cupo", width: 20, align: "right" },
    { key: "inscritos", label: "Inscritos", width: 20, align: "right" },
    { key: "disponibles", label: "Disponibles", width: 20, align: "right" },
  ];

  const mappedRows: string[][] = rows.map((row) => [
    programName || "—",
    String(row.quotaCapacity),
    String(row.enrolledCount),
    String(row.availableSeats),
  ]);

  const truncatedTo = truncated ? PROGRAM_SUMMARY_TRUNCATION_CAP : undefined;
  const footer = truncated
    ? `Documento truncado a ${PROGRAM_SUMMARY_TRUNCATION_CAP} filas`
    : `Reporte generado el ${generatedAt}`;

  return {
    title: "Resumen de Programa",
    appliedFilter: filterLabel,
    generatedAt,
    truncatedTo,
    columns,
    rows: mappedRows,
    footer,
  };
}

export { PROGRAM_SUMMARY_TRUNCATION_CAP };
