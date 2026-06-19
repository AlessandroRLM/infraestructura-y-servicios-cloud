/**
 * Unit tests for toSectionGradeReportModel mapper.
 * Pure function — no render, no transport.
 * Covers AC-3.d, AC-4.f, RF-9.1.
 */
import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import {
  GetSectionGradeReportResponseSchema,
  PartialGradeSchema,
  StudentGradeRowSchema,
} from "@/gen/reports/v1/reports_pb";
import {
  SECTION_GRADE_TRUNCATION_CAP,
  toSectionGradeReportModel,
} from "../pdf/toSectionGradeReportModel";

function makeRow(overrides: {
  givenNames?: string;
  lastNamePaternal?: string;
  lastNameMaternal?: string;
  partialGrades?: { position: number; value: string }[];
  finalGrade?: string;
  outcome?: string;
}) {
  return create(StudentGradeRowSchema, {
    studentId: "s1",
    givenNames: overrides.givenNames ?? "Ana",
    lastNamePaternal: overrides.lastNamePaternal ?? "García",
    lastNameMaternal: overrides.lastNameMaternal ?? "López",
    partialGrades: (overrides.partialGrades ?? []).map((pg) =>
      create(PartialGradeSchema, {
        evaluationId: `eval-${pg.position}`,
        position: pg.position,
        value: pg.value,
      }),
    ),
    finalGrade: overrides.finalGrade ?? "",
    outcome: overrides.outcome ?? "in_progress",
  });
}

function makeResponse(overrides: {
  sectionId?: string;
  rows?: ReturnType<typeof makeRow>[];
  generatedAt?: string;
  truncated?: boolean;
}) {
  return create(GetSectionGradeReportResponseSchema, {
    sectionId: overrides.sectionId ?? "section-uuid-1234",
    rows: overrides.rows ?? [],
    generatedAt: overrides.generatedAt ?? "2026-06-18T10:00:00Z",
    truncated: overrides.truncated ?? false,
  });
}

describe("toSectionGradeReportModel", () => {
  it("AC-3.d: maps columns correctly — Alumno + evaluation columns + Final + Resultado", () => {
    const response = makeResponse({
      rows: [
        makeRow({
          partialGrades: [
            { position: 1, value: "5.5" },
            { position: 2, value: "6.0" },
          ],
          finalGrade: "5.75",
          outcome: "passed",
        }),
      ],
    });

    const model = toSectionGradeReportModel(response, "Sección A");

    const keys = model.columns.map((c) => c.key);
    expect(keys[0]).toBe("alumno");
    expect(keys).toContain("eval_1");
    expect(keys).toContain("eval_2");
    expect(keys[keys.length - 2]).toBe("final");
    expect(keys[keys.length - 1]).toBe("resultado");
  });

  it("AC-3.d: maps rows correctly — full name, partial grades, final, outcome", () => {
    const response = makeResponse({
      rows: [
        makeRow({
          givenNames: "María",
          lastNamePaternal: "Rodríguez",
          lastNameMaternal: "Soto",
          partialGrades: [
            { position: 1, value: "5.5" },
            { position: 2, value: "6.0" },
          ],
          finalGrade: "5.75",
          outcome: "passed",
        }),
      ],
    });

    const model = toSectionGradeReportModel(response, "Sección A");

    expect(model.rows).toHaveLength(1);
    const row = model.rows[0]!;
    expect(row[0]).toBe("María Rodríguez Soto");
    expect(row).toContain("5.5");
    expect(row).toContain("6.0");
    expect(row).toContain("5.75");
    expect(row).toContain("Aprobado");
  });

  it("AC-3.d: generatedAt is preserved in the model", () => {
    const response = makeResponse({ generatedAt: "2026-06-18T10:00:00Z" });
    const model = toSectionGradeReportModel(response, "Sección B");

    expect(model.generatedAt).toBe("2026-06-18T10:00:00Z");
  });

  it("AC-4.f: truncated=true → truncatedTo equals SECTION_GRADE_TRUNCATION_CAP (500)", () => {
    const response = makeResponse({ truncated: true });
    const model = toSectionGradeReportModel(response, "Sección C");

    expect(model.truncatedTo).toBe(SECTION_GRADE_TRUNCATION_CAP);
    expect(model.truncatedTo).toBe(500);
  });

  it("truncated=false → truncatedTo is undefined", () => {
    const response = makeResponse({ truncated: false });
    const model = toSectionGradeReportModel(response, "Sección D");

    expect(model.truncatedTo).toBeUndefined();
  });

  it("RF-4.5: truncated=true → footer contains 'truncado a 500 filas'", () => {
    const response = makeResponse({ truncated: true });
    const model = toSectionGradeReportModel(response, "Sección E");

    expect(model.footer).toMatch(/truncado a 500 filas/i);
  });

  it("title is 'Calificaciones por Sección'", () => {
    const model = toSectionGradeReportModel(makeResponse({}), "Sección F");
    expect(model.title).toBe("Calificaciones por Sección");
  });

  it("appliedFilter uses the provided sectionLabel", () => {
    const model = toSectionGradeReportModel(
      makeResponse({}),
      "Sección Custom Label",
    );
    expect(model.appliedFilter).toBe("Sección Custom Label");
  });

  it("appliedFilter falls back to short sectionId when label is empty", () => {
    const response = makeResponse({ sectionId: "abcd1234-5678-90ef" });
    const model = toSectionGradeReportModel(response, "");
    expect(model.appliedFilter).toContain("abcd1234");
  });

  it("missing partial grade for a position renders '—'", () => {
    // Force position 2 to appear by adding another row that has it.
    const response2 = makeResponse({
      rows: [
        makeRow({
          partialGrades: [
            { position: 1, value: "5.0" },
            { position: 2, value: "" },
          ],
          finalGrade: "5.0",
          outcome: "passed",
        }),
        makeRow({
          givenNames: "Pedro",
          lastNamePaternal: "López",
          lastNameMaternal: "",
          partialGrades: [
            { position: 1, value: "6.0" },
            // position 2 absent for this student
          ],
          finalGrade: "6.0",
          outcome: "passed",
        }),
      ],
    });

    const model = toSectionGradeReportModel(response2, "Sección G");
    const pedro = model.rows[1]!;
    // Eval position 2 should be "—" for Pedro since he has no grade there.
    const evalColIdx = model.columns.findIndex((c) => c.key === "eval_2");
    expect(pedro[evalColIdx]).toBe("—");
  });

  it("outcome 'failed' maps to 'Reprobado'", () => {
    const response = makeResponse({
      rows: [makeRow({ outcome: "failed", finalGrade: "3.0" })],
    });
    const model = toSectionGradeReportModel(response, "Sección H");
    const row = model.rows[0]!;
    expect(row[row.length - 1]).toBe("Reprobado");
  });

  it("outcome 'in_progress' maps to 'En curso'", () => {
    const response = makeResponse({
      rows: [makeRow({ outcome: "in_progress" })],
    });
    const model = toSectionGradeReportModel(response, "Sección I");
    const row = model.rows[0]!;
    expect(row[row.length - 1]).toBe("En curso");
  });

  it("column widths sum to 100", () => {
    const response = makeResponse({
      rows: [
        makeRow({
          partialGrades: [
            { position: 1, value: "5.5" },
            { position: 2, value: "6.0" },
            { position: 3, value: "5.0" },
          ],
        }),
      ],
    });
    const model = toSectionGradeReportModel(response, "Sección J");
    const total = model.columns.reduce((s, c) => s + c.width, 0);
    expect(total).toBe(100);
  });

  it("empty rows → model has empty rows array", () => {
    const response = makeResponse({ rows: [] });
    const model = toSectionGradeReportModel(response, "Sección K");
    expect(model.rows).toHaveLength(0);
  });

  it("partial grade value '0' is NOT coerced to '—' ('0' is truthy so || preserves it)", () => {
    // "0" || "—" === "0" — the string "0" is truthy in JS; the fallback is never reached.
    const response = makeResponse({
      rows: [
        makeRow({
          partialGrades: [{ position: 1, value: "0" }],
          finalGrade: "0",
          outcome: "failed",
        }),
      ],
    });
    const model = toSectionGradeReportModel(response, "Sección L");
    const row = model.rows[0]!;
    const evalColIdx = model.columns.findIndex((c) => c.key === "eval_1");
    expect(row[evalColIdx]).toBe("0");
  });

  it("partial grade value '' (proto3 default for absent grade) renders as '—'", () => {
    // proto3 sends "" for an unset string field; || "—" converts "" to "—" (empty is falsy).
    const response = makeResponse({
      rows: [
        makeRow({
          partialGrades: [{ position: 1, value: "" }],
          finalGrade: "5.0",
          outcome: "passed",
        }),
      ],
    });
    const model = toSectionGradeReportModel(response, "Sección M");
    const row = model.rows[0]!;
    const evalColIdx = model.columns.findIndex((c) => c.key === "eval_1");
    expect(row[evalColIdx]).toBe("—");
  });

  it("finalGrade '' (proto3 default for absent final grade) renders as '—'", () => {
    // proto3 sends "" for an unset finalGrade; || "—" converts "" to "—".
    const response = makeResponse({
      rows: [
        makeRow({
          partialGrades: [{ position: 1, value: "5.0" }],
          finalGrade: "",
          outcome: "in_progress",
        }),
      ],
    });
    const model = toSectionGradeReportModel(response, "Sección N");
    const row = model.rows[0]!;
    const finalColIdx = model.columns.findIndex((c) => c.key === "final");
    expect(row[finalColIdx]).toBe("—");
  });
});
