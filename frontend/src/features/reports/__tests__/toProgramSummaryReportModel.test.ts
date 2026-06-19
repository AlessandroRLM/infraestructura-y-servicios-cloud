/**
 * Unit tests for toProgramSummaryReportModel mapper.
 * Pure function — no render, no transport.
 * Covers AC-3.d, AC-4.f, RF-9.1 for the program summary report.
 */
import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import {
  GetProgramSummaryReportResponseSchema,
  ProgramEnrollmentRowSchema,
} from "@/gen/reports/v1/reports_pb";
import {
  PROGRAM_SUMMARY_TRUNCATION_CAP,
  toProgramSummaryReportModel,
} from "../pdf/toProgramSummaryReportModel";

function makeRow(overrides: {
  quotaId?: string;
  quotaCapacity?: number;
  enrolledCount?: number;
  availableSeats?: number;
  fillPercentage?: string;
}) {
  return create(ProgramEnrollmentRowSchema, {
    quotaId: overrides.quotaId ?? "quota-uuid-1",
    quotaCapacity: overrides.quotaCapacity ?? 100,
    enrolledCount: overrides.enrolledCount ?? 80,
    availableSeats: overrides.availableSeats ?? 20,
    fillPercentage: overrides.fillPercentage ?? "80.0",
  });
}

function makeResponse(overrides: {
  programId?: string;
  programName?: string;
  year?: number;
  rows?: ReturnType<typeof makeRow>[];
  generatedAt?: string;
  truncated?: boolean;
}) {
  return create(GetProgramSummaryReportResponseSchema, {
    programId: overrides.programId ?? "program-uuid-1",
    programName: overrides.programName ?? "Ingeniería Civil",
    year: overrides.year ?? 2026,
    rows: overrides.rows ?? [],
    generatedAt: overrides.generatedAt ?? "2026-06-18T10:00:00Z",
    truncated: overrides.truncated ?? false,
  });
}

describe("toProgramSummaryReportModel", () => {
  it("produces correct columns — Programa, Cupo, Inscritos, Disponibles", () => {
    const response = makeResponse({ rows: [makeRow({})] });
    const model = toProgramSummaryReportModel(response, "fallback");

    expect(model.columns.map((c) => c.key)).toEqual([
      "programa",
      "cupo",
      "inscritos",
      "disponibles",
    ]);
    expect(model.columns.map((c) => c.label)).toEqual([
      "Programa",
      "Cupo",
      "Inscritos",
      "Disponibles",
    ]);
  });

  it("numeric columns use align:right; text column uses align:left", () => {
    const response = makeResponse({ rows: [makeRow({})] });
    const model = toProgramSummaryReportModel(response, "fallback");

    const aligns = model.columns.map((c) => c.align);
    expect(aligns[0]).toBe("left"); // Programa
    expect(aligns[1]).toBe("right"); // Cupo
    expect(aligns[2]).toBe("right"); // Inscritos
    expect(aligns[3]).toBe("right"); // Disponibles
  });

  it("column widths sum to 100", () => {
    const response = makeResponse({ rows: [makeRow({})] });
    const model = toProgramSummaryReportModel(response, "fallback");
    const sum = model.columns.reduce((s, c) => s + c.width, 0);
    expect(sum).toBe(100);
  });

  it("maps rows correctly — programName, quotaCapacity, enrolledCount, availableSeats", () => {
    const row = makeRow({
      quotaCapacity: 120,
      enrolledCount: 95,
      availableSeats: 25,
    });
    const response = makeResponse({
      programName: "Medicina",
      rows: [row],
    });
    const model = toProgramSummaryReportModel(response, "fallback");

    expect(model.rows).toHaveLength(1);
    const [mapped] = model.rows;
    expect(mapped![0]).toBe("Medicina"); // programName used for each row
    expect(mapped![1]).toBe("120");
    expect(mapped![2]).toBe("95");
    expect(mapped![3]).toBe("25");
  });

  it("appliedFilter uses programName + year from response", () => {
    const response = makeResponse({ programName: "Derecho", year: 2025 });
    const model = toProgramSummaryReportModel(response, "fallback");
    expect(model.appliedFilter).toBe("Derecho — 2025");
  });

  it("generatedAt is passed through to the model", () => {
    const response = makeResponse({ generatedAt: "2026-06-18T15:00:00Z" });
    const model = toProgramSummaryReportModel(response, "prog");
    expect(model.generatedAt).toBe("2026-06-18T15:00:00Z");
  });

  it("AC-4.f: truncated=false → truncatedTo is undefined, footer is generation line", () => {
    const response = makeResponse({ truncated: false });
    const model = toProgramSummaryReportModel(response, "prog");
    expect(model.truncatedTo).toBeUndefined();
    expect(model.footer).toMatch(/Reporte generado/);
    expect(model.footer).not.toMatch(/truncado/);
  });

  it("AC-4.f: truncated=true → truncatedTo=200, footer contains truncation notice", () => {
    const response = makeResponse({ truncated: true });
    const model = toProgramSummaryReportModel(response, "prog");
    expect(model.truncatedTo).toBe(PROGRAM_SUMMARY_TRUNCATION_CAP);
    expect(model.truncatedTo).toBe(200);
    expect(model.footer).toMatch(/truncado a 200 filas/);
  });

  it("title is 'Resumen de Programa'", () => {
    const response = makeResponse({});
    const model = toProgramSummaryReportModel(response, "prog");
    expect(model.title).toBe("Resumen de Programa");
  });
});
