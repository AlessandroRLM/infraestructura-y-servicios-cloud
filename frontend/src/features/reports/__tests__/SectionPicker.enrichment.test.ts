/**
 * SectionPicker enrichment tests — buildSectionLabel pure helper.
 * Validates that the enriched label renders "CODE · Name — Year · Semestre N"
 * with graceful fallback when course or period data is missing.
 */
import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import {
  AcademicPeriodSchema,
  CourseSchema,
  SectionSchema,
} from "@/gen/catalog/v1/catalog_pb";
import { buildSectionLabel } from "../components/SectionPicker";

function makeSection(courseId: string, academicPeriodId: string) {
  return create(SectionSchema, {
    id: "abcdef12-3456-7890-abcd-ef1234567890",
    courseId,
    academicPeriodId,
    seatCapacity: 30,
    createdAt: "",
    updatedAt: "",
  });
}

function makeCourse(id: string, code: string, name: string) {
  return create(CourseSchema, {
    id,
    code,
    name,
    credits: 4,
    createdAt: "",
    updatedAt: "",
  });
}

function makePeriod(id: string, year: number, term: number) {
  return create(AcademicPeriodSchema, {
    id,
    year,
    term,
    startDate: "",
    endDate: "",
    createdAt: "",
    updatedAt: "",
  });
}

describe("buildSectionLabel", () => {
  it("returns full enriched label when both course and period are available", () => {
    const section = makeSection("course-1", "period-1");
    const courseMap = new Map([
      ["course-1", makeCourse("course-1", "MAT101", "Cálculo I")],
    ]);
    const periodMap = new Map([["period-1", makePeriod("period-1", 2026, 1)]]);

    const label = buildSectionLabel(section, courseMap, periodMap);
    expect(label).toBe("MAT101 · Cálculo I — 2026 · Semestre 1");
  });

  it("returns course-only label when period is missing", () => {
    const section = makeSection("course-1", "period-missing");
    const courseMap = new Map([
      ["course-1", makeCourse("course-1", "QUI201", "Química General")],
    ]);
    const periodMap = new Map<string, ReturnType<typeof makePeriod>>();

    const label = buildSectionLabel(section, courseMap, periodMap);
    expect(label).toBe("QUI201 · Química General");
  });

  it("falls back to short ID when neither course nor period is available", () => {
    const section = makeSection("course-missing", "period-missing");
    const courseMap = new Map<string, ReturnType<typeof makeCourse>>();
    const periodMap = new Map<string, ReturnType<typeof makePeriod>>();

    const label = buildSectionLabel(section, courseMap, periodMap);
    // Should show a truncated section id
    expect(label).toMatch(/^Sección abcdef12/);
    expect(label).toContain("…");
  });

  it("uses term 2 correctly in the label", () => {
    const section = makeSection("course-1", "period-2");
    const courseMap = new Map([
      ["course-1", makeCourse("course-1", "FIS301", "Física III")],
    ]);
    const periodMap = new Map([["period-2", makePeriod("period-2", 2025, 2)]]);

    const label = buildSectionLabel(section, courseMap, periodMap);
    expect(label).toBe("FIS301 · Física III — 2025 · Semestre 2");
  });
});
