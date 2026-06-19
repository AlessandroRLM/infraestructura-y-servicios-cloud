import type { GetStudentRecordReportResponse } from "@/gen/reports/v1/reports_pb";
import type { ReportPdfModel } from "./model";

/** Truncation cap for StudentRecord as specified in RF-4.6. */
const STUDENT_RECORD_TRUNCATION_CAP = 1000;

/**
 * Maps a GetStudentRecordReportResponse to the normalized ReportPdfModel.
 *
 * Column layout:
 *   - Período (academic_period_name) — text, left
 *   - Curso (course_name)            — text, left
 *   - Estado (enrollment_status)     — text, left
 *   - Nota (final_grade)             — numeric/grade, right
 *
 * Column widths sum to 100 (%).
 * When truncated is true, sets truncatedTo = 1000 and bakes notice into footer.
 *
 * The mapper is pure — no hooks, no side effects, safe to unit-test directly.
 */
export function toStudentRecordReportModel(
  response: GetStudentRecordReportResponse,
): ReportPdfModel {
  const { rows, generatedAt, truncated, studentName, studentId } = response;

  // Use studentName from the response when available; fall back to studentId.
  const filterLabel = studentName || studentId;

  const columns: ReportPdfModel["columns"] = [
    { key: "periodo", label: "Período", width: 25, align: "left" },
    { key: "curso", label: "Curso", width: 40, align: "left" },
    { key: "estado", label: "Estado", width: 20, align: "left" },
    { key: "nota", label: "Nota", width: 15, align: "right" },
  ];

  const mappedRows: string[][] = rows.map((row) => [
    row.academicPeriodName || "—",
    row.courseName || "—",
    formatEnrollmentStatus(row.enrollmentStatus),
    row.finalGrade || "—",
  ]);

  const truncatedTo = truncated ? STUDENT_RECORD_TRUNCATION_CAP : undefined;
  const footer = truncated
    ? `Documento truncado a ${STUDENT_RECORD_TRUNCATION_CAP} filas`
    : `Reporte generado el ${generatedAt}`;

  return {
    title: "Expediente de Alumno",
    appliedFilter: filterLabel,
    generatedAt,
    truncatedTo,
    columns,
    rows: mappedRows,
    footer,
  };
}

/**
 * Maps a machine enrollment_status string to a human-readable Spanish label.
 */
function formatEnrollmentStatus(status: string): string {
  switch (status) {
    case "passed":
      return "Aprobado";
    case "failed":
      return "Reprobado";
    case "in_progress":
      return "En curso";
    case "withdrawn":
      return "Retirado";
    default:
      return status || "—";
  }
}

export { STUDENT_RECORD_TRUNCATION_CAP };
