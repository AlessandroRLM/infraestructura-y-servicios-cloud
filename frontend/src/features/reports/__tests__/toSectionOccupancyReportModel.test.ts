/**
 * Unit tests for toSectionOccupancyReportModel mapper.
 * Pure function — no render, no transport.
 * Covers AC-3.d, AC-4.f, RF-9.1 for the occupancy report.
 */
import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import {
  GetSectionOccupancyReportResponseSchema,
  SectionOccupancyRowSchema,
} from "@/gen/reports/v1/reports_pb";
import {
  SECTION_OCCUPANCY_TRUNCATION_CAP,
  toSectionOccupancyReportModel,
} from "../pdf/toSectionOccupancyReportModel";

function makeRow(overrides: {
  sectionId?: string;
  courseName?: string;
  capacity?: number;
  activeSeatCount?: number;
  fillPercentage?: string;
}) {
  return create(SectionOccupancyRowSchema, {
    sectionId: overrides.sectionId ?? "section-uuid-123456789",
    courseName: overrides.courseName ?? "Cálculo I",
    capacity: overrides.capacity ?? 40,
    activeSeatCount: overrides.activeSeatCount ?? 30,
    fillPercentage: overrides.fillPercentage ?? "75.0",
  });
}

function makeResponse(overrides: {
  academicPeriodId?: string;
  rows?: ReturnType<typeof makeRow>[];
  generatedAt?: string;
  truncated?: boolean;
  academicPeriodName?: string;
}) {
  return create(GetSectionOccupancyReportResponseSchema, {
    academicPeriodId: overrides.academicPeriodId ?? "period-uuid-1",
    rows: overrides.rows ?? [],
    generatedAt: overrides.generatedAt ?? "2026-06-18T10:00:00Z",
    truncated: overrides.truncated ?? false,
    academicPeriodName: overrides.academicPeriodName ?? "2026 · Semestre 1",
  });
}

describe("toSectionOccupancyReportModel", () => {
  it("produces correct columns — Curso, Sección, Cupo, Inscritos, % Ocupación", () => {
    const response = makeResponse({ rows: [makeRow({})] });
    const model = toSectionOccupancyReportModel(response, "fallback");

    expect(model.columns.map((c) => c.key)).toEqual([
      "curso",
      "seccion",
      "cupo",
      "inscritos",
      "ocupacion",
    ]);
    expect(model.columns.map((c) => c.label)).toEqual([
      "Curso",
      "Sección",
      "Cupo",
      "Inscritos",
      "% Ocupación",
    ]);
  });

  it("numeric columns use align:right; text columns use align:left", () => {
    const response = makeResponse({ rows: [makeRow({})] });
    const model = toSectionOccupancyReportModel(response, "fallback");

    const aligns = model.columns.map((c) => c.align);
    expect(aligns[0]).toBe("left"); // Curso
    expect(aligns[1]).toBe("left"); // Sección
    expect(aligns[2]).toBe("right"); // Cupo
    expect(aligns[3]).toBe("right"); // Inscritos
    expect(aligns[4]).toBe("right"); // % Ocupación
  });

  it("column widths sum to 100", () => {
    const response = makeResponse({ rows: [makeRow({})] });
    const model = toSectionOccupancyReportModel(response, "fallback");
    const sum = model.columns.reduce((s, c) => s + c.width, 0);
    expect(sum).toBe(100);
  });

  it("maps rows correctly — course name, section short id, capacity, enrolled, fill%", () => {
    const row = makeRow({
      sectionId: "abcdef12-ghij-klmn",
      courseName: "Álgebra",
      capacity: 50,
      activeSeatCount: 45,
      fillPercentage: "90.0",
    });
    const response = makeResponse({ rows: [row] });
    const model = toSectionOccupancyReportModel(response, "fallback");

    expect(model.rows).toHaveLength(1);
    const [mapped] = model.rows;
    expect(mapped![0]).toBe("Álgebra");
    expect(mapped![1]).toMatch(/^abcdef12/);
    expect(mapped![2]).toBe("50");
    expect(mapped![3]).toBe("45");
    expect(mapped![4]).toBe("90.0%");
  });

  it("uses academicPeriodName from response as appliedFilter when available", () => {
    const response = makeResponse({ academicPeriodName: "2026 · Semestre 1" });
    const model = toSectionOccupancyReportModel(response, "fallback-label");
    expect(model.appliedFilter).toBe("2026 · Semestre 1");
  });

  it("falls back to caller-supplied periodLabel when academicPeriodName is empty", () => {
    const response = makeResponse({ academicPeriodName: "" });
    const model = toSectionOccupancyReportModel(response, "My fallback label");
    expect(model.appliedFilter).toBe("My fallback label");
  });

  it("generatedAt is passed through to the model", () => {
    const response = makeResponse({ generatedAt: "2026-06-18T12:30:00Z" });
    const model = toSectionOccupancyReportModel(response, "period");
    expect(model.generatedAt).toBe("2026-06-18T12:30:00Z");
  });

  it("AC-4.f: truncated=false → truncatedTo is undefined, footer is generation line", () => {
    const response = makeResponse({ truncated: false });
    const model = toSectionOccupancyReportModel(response, "period");
    expect(model.truncatedTo).toBeUndefined();
    expect(model.footer).toMatch(/Reporte generado/);
    expect(model.footer).not.toMatch(/truncado/);
  });

  it("AC-4.f: truncated=true → truncatedTo=1000, footer contains truncation notice", () => {
    const response = makeResponse({ truncated: true });
    const model = toSectionOccupancyReportModel(response, "period");
    expect(model.truncatedTo).toBe(SECTION_OCCUPANCY_TRUNCATION_CAP);
    expect(model.truncatedTo).toBe(1000);
    expect(model.footer).toMatch(/truncado a 1000 filas/);
  });

  it("title is 'Ocupación por Período'", () => {
    const response = makeResponse({});
    const model = toSectionOccupancyReportModel(response, "period");
    expect(model.title).toBe("Ocupación por Período");
  });
});
