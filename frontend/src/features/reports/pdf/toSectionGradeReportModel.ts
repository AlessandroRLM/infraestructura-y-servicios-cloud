import type { GetSectionGradeReportResponse } from "@/gen/reports/v1/reports_pb";
import type { ReportPdfModel } from "./model";

/** Truncation cap for SectionGrade as specified in RF-4.6. */
const SECTION_GRADE_TRUNCATION_CAP = 500;

/**
 * Maps a GetSectionGradeReportResponse to the normalized ReportPdfModel.
 *
 * Column layout:
 *   - Alumno (student full name)
 *   - Evaluación N for each position found in the response (ordered by position ASC)
 *   - Final (finalGrade or "—" when absent)
 *   - Resultado (outcome mapped to a readable label)
 *
 * When truncated is true, sets truncatedTo = SECTION_GRADE_TRUNCATION_CAP and
 * adds the stamp to the footer so the in-PDF notice is baked into the document.
 *
 * The mapper is pure — no hooks, no side effects, safe to unit-test directly.
 */
export function toSectionGradeReportModel(
  response: GetSectionGradeReportResponse,
  /** Human-readable filter label shown in the appliedFilter line (e.g. the section identifier). */
  sectionLabel: string,
): ReportPdfModel {
  const { rows, generatedAt, truncated, sectionId } = response;

  // Derive evaluation positions from the response rows to build dynamic columns.
  // Collect all unique positions (ordered ASC) across all students.
  const positionSet = new Set<number>();
  for (const row of rows) {
    for (const pg of row.partialGrades) {
      positionSet.add(pg.position);
    }
  }
  const positions = Array.from(positionSet).sort((a, b) => a - b);

  // Build column descriptors. Widths are relative percentages summing to 100.
  // Base: Alumno (30), each evaluation (10), Final (10), Resultado (10).
  // If no evaluations, Alumno takes more space.
  const evalCount = positions.length;
  // Distribute widths roughly evenly (integers sum to 100).
  const evalWidth = evalCount > 0 ? Math.floor(50 / evalCount) : 0;
  const alumnoWidth = 100 - evalCount * evalWidth - 10 - 10; // 10 for Final, 10 for Resultado

  const columns: ReportPdfModel["columns"] = [
    {
      key: "alumno",
      label: "Alumno",
      width: Math.max(alumnoWidth, 20),
      align: "left",
    },
    ...positions.map((pos) => ({
      key: `eval_${pos}`,
      label: `Evaluación ${pos}`,
      width: evalWidth > 0 ? evalWidth : 10,
      align: "right" as const,
    })),
    { key: "final", label: "Final", width: 10, align: "right" as const },
    { key: "resultado", label: "Resultado", width: 10, align: "left" as const },
  ];

  // Normalize column widths so they sum to exactly 100.
  const rawSum = columns.reduce((s, c) => s + c.width, 0);
  if (rawSum !== 100 && columns.length > 0) {
    const last = columns[columns.length - 1]!;
    last.width = last.width + (100 - rawSum);
  }

  // Map each StudentGradeRow to a string row matching the column order.
  const mappedRows: string[][] = rows.map((row) => {
    const fullName = [
      row.givenNames,
      row.lastNamePaternal,
      row.lastNameMaternal,
    ]
      .filter(Boolean)
      .join(" ");

    const evalCells = positions.map((pos) => {
      const pg = row.partialGrades.find((g) => g.position === pos);
      return pg?.value || "—";
    });

    const final = row.finalGrade || "—";
    const resultado = mapOutcome(row.outcome);

    return [fullName, ...evalCells, final, resultado];
  });

  const filterLabel = sectionLabel || `Sección ${sectionId.slice(0, 8)}`;

  const truncatedTo = truncated ? SECTION_GRADE_TRUNCATION_CAP : undefined;
  const footer = truncated
    ? `Documento truncado a ${SECTION_GRADE_TRUNCATION_CAP} filas`
    : `Reporte generado el ${generatedAt}`;

  return {
    title: "Calificaciones por Sección",
    appliedFilter: filterLabel,
    generatedAt,
    truncatedTo,
    columns,
    rows: mappedRows,
    footer,
  };
}

/** Maps an outcome proto string to a readable Spanish label. */
function mapOutcome(outcome: string): string {
  switch (outcome) {
    case "passed":
      return "Aprobado";
    case "failed":
      return "Reprobado";
    case "in_progress":
      return "En curso";
    default:
      return outcome || "—";
  }
}

// Re-export the cap for tests that assert the truncation value.
export { SECTION_GRADE_TRUNCATION_CAP };
