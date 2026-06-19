/**
 * Unit tests for toStudentRecordReportModel mapper.
 * Pure function — no render, no transport.
 * Covers AC-3.d, AC-4.f, RF-9.1 for the student-record report.
 */
import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import {
  AcademicRecordRowSchema,
  GetStudentRecordReportResponseSchema,
} from "@/gen/reports/v1/reports_pb";
import {
  STUDENT_RECORD_TRUNCATION_CAP,
  toStudentRecordReportModel,
} from "../pdf/toStudentRecordReportModel";

function makeRow(overrides: {
  academicPeriodName?: string;
  courseName?: string;
  enrollmentStatus?: string;
  finalGrade?: string;
}) {
  return create(AcademicRecordRowSchema, {
    academicPeriodId: "period-uuid-1",
    academicPeriodName: overrides.academicPeriodName ?? "2026 · Semestre 1",
    sectionId: "section-uuid-1",
    courseName: overrides.courseName ?? "Cálculo I",
    enrollmentStatus: overrides.enrollmentStatus ?? "passed",
    finalGrade: overrides.finalGrade ?? "6.5",
    outcome: "passed",
  });
}

function makeResponse(overrides: {
  studentId?: string;
  studentName?: string;
  rows?: ReturnType<typeof makeRow>[];
  generatedAt?: string;
  truncated?: boolean;
}) {
  return create(GetStudentRecordReportResponseSchema, {
    studentId: overrides.studentId ?? "student-uuid-1",
    studentName: overrides.studentName ?? "Ana García",
    rows: overrides.rows ?? [],
    generatedAt: overrides.generatedAt ?? "2026-06-18T10:00:00Z",
    truncated: overrides.truncated ?? false,
  });
}

describe("toStudentRecordReportModel", () => {
  it("produces correct columns — Período, Curso, Estado, Nota", () => {
    const response = makeResponse({ rows: [makeRow({})] });
    const model = toStudentRecordReportModel(response);

    expect(model.columns.map((c) => c.key)).toEqual([
      "periodo",
      "curso",
      "estado",
      "nota",
    ]);
    expect(model.columns.map((c) => c.label)).toEqual([
      "Período",
      "Curso",
      "Estado",
      "Nota",
    ]);
  });

  it("Nota column uses align:right; all others use align:left", () => {
    const response = makeResponse({ rows: [makeRow({})] });
    const model = toStudentRecordReportModel(response);

    const aligns = model.columns.map((c) => c.align);
    expect(aligns[0]).toBe("left"); // Período
    expect(aligns[1]).toBe("left"); // Curso
    expect(aligns[2]).toBe("left"); // Estado
    expect(aligns[3]).toBe("right"); // Nota
  });

  it("column widths sum to 100", () => {
    const response = makeResponse({ rows: [makeRow({})] });
    const model = toStudentRecordReportModel(response);
    const sum = model.columns.reduce((s, c) => s + c.width, 0);
    expect(sum).toBe(100);
  });

  it("maps rows correctly — period, course, status label, grade", () => {
    const row = makeRow({
      academicPeriodName: "2025 · Semestre 2",
      courseName: "Álgebra",
      enrollmentStatus: "failed",
      finalGrade: "3.2",
    });
    const response = makeResponse({ rows: [row] });
    const model = toStudentRecordReportModel(response);

    expect(model.rows).toHaveLength(1);
    const [mapped] = model.rows;
    expect(mapped![0]).toBe("2025 · Semestre 2");
    expect(mapped![1]).toBe("Álgebra");
    expect(mapped![2]).toBe("Reprobado");
    expect(mapped![3]).toBe("3.2");
  });

  it("maps enrollment statuses to Spanish labels", () => {
    const statuses = [
      { input: "passed", expected: "Aprobado" },
      { input: "failed", expected: "Reprobado" },
      { input: "in_progress", expected: "En curso" },
      { input: "withdrawn", expected: "Retirado" },
    ];

    for (const { input, expected } of statuses) {
      const row = makeRow({ enrollmentStatus: input });
      const response = makeResponse({ rows: [row] });
      const model = toStudentRecordReportModel(response);
      expect(model.rows[0]![2]).toBe(expected);
    }
  });

  it("uses studentName from response as appliedFilter when available", () => {
    const response = makeResponse({ studentName: "María López" });
    const model = toStudentRecordReportModel(response);
    expect(model.appliedFilter).toBe("María López");
  });

  it("falls back to studentId when studentName is empty", () => {
    const response = makeResponse({
      studentName: "",
      studentId: "student-uuid-fallback",
    });
    const model = toStudentRecordReportModel(response);
    expect(model.appliedFilter).toBe("student-uuid-fallback");
  });

  it("generatedAt is passed through to the model", () => {
    const response = makeResponse({ generatedAt: "2026-06-18T12:30:00Z" });
    const model = toStudentRecordReportModel(response);
    expect(model.generatedAt).toBe("2026-06-18T12:30:00Z");
  });

  it("AC-4.f: truncated=false → truncatedTo is undefined, footer is generation line", () => {
    const response = makeResponse({ truncated: false });
    const model = toStudentRecordReportModel(response);
    expect(model.truncatedTo).toBeUndefined();
    expect(model.footer).toMatch(/Reporte generado/);
    expect(model.footer).not.toMatch(/truncado/);
  });

  it("AC-4.f: truncated=true → truncatedTo=1000, footer contains truncation notice", () => {
    const response = makeResponse({ truncated: true });
    const model = toStudentRecordReportModel(response);
    expect(model.truncatedTo).toBe(STUDENT_RECORD_TRUNCATION_CAP);
    expect(model.truncatedTo).toBe(1000);
    expect(model.footer).toMatch(/truncado a 1000 filas/);
  });

  it("title is 'Expediente de Alumno'", () => {
    const response = makeResponse({});
    const model = toStudentRecordReportModel(response);
    expect(model.title).toBe("Expediente de Alumno");
  });
});
