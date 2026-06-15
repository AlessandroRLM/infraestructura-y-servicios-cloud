import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import { OwnGradeSchema } from "@/gen/grades/v1/grades_pb";
import {
  formatPeriod,
  formatStatus,
  formatWeight,
  groupBySection,
  STATUS_LABELS,
} from "../groupBySection";

function makeGrade(
  overrides: Partial<Parameters<typeof create<typeof OwnGradeSchema>>[1]> = {},
) {
  return create(OwnGradeSchema, {
    id: "grade-1",
    evaluationId: "eval-1",
    sectionEnrollmentId: "se-1",
    value: "5.0",
    version: 1,
    createdAt: "",
    updatedAt: "",
    courseCode: "MAT101",
    courseName: "Matemáticas",
    evaluationPosition: 1,
    evaluationWeight: "0.500",
    sectionFinalGrade: "5.5",
    sectionStatus: "passed",
    academicPeriodYear: 2026,
    academicPeriodTerm: 1,
    programId: "prog-1",
    programName: "Ingeniería Civil",
    ...overrides,
  });
}

describe("formatPeriod", () => {
  it("formats as {year}-{term}", () => {
    expect(formatPeriod(2026, 1)).toBe("2026-1");
    expect(formatPeriod(2025, 2)).toBe("2025-2");
  });
});

describe("formatStatus", () => {
  it("maps all known raw statuses to Spanish", () => {
    expect(formatStatus("in_progress")).toBe("En curso");
    expect(formatStatus("passed")).toBe("Aprobado");
    expect(formatStatus("failed")).toBe("Reprobado");
    expect(formatStatus("withdrawn")).toBe("Retirado");
  });

  it("passes through unknown status values unchanged", () => {
    expect(formatStatus("unknown_status")).toBe("unknown_status");
  });

  it("STATUS_LABELS covers all four expected keys", () => {
    expect(Object.keys(STATUS_LABELS)).toEqual(
      expect.arrayContaining(["in_progress", "passed", "failed", "withdrawn"]),
    );
  });
});

describe("formatWeight", () => {
  it("converts decimal strings to rounded integer percentages", () => {
    expect(formatWeight("0.300")).toBe("30%");
    expect(formatWeight("0.500")).toBe("50%");
    expect(formatWeight("0.400")).toBe("40%");
    expect(formatWeight("1.000")).toBe("100%");
  });

  it("rounds to nearest integer percent", () => {
    expect(formatWeight("0.333")).toBe("33%");
    expect(formatWeight("0.667")).toBe("67%");
  });
});

describe("groupBySection", () => {
  it("S-F2a: groups two grades for same section into one group", () => {
    const grades = [
      makeGrade({
        id: "g1",
        evaluationId: "e1",
        evaluationPosition: 1,
        evaluationWeight: "0.500",
        value: "5.0",
      }),
      makeGrade({
        id: "g2",
        evaluationId: "e2",
        evaluationPosition: 2,
        evaluationWeight: "0.500",
        value: "6.0",
      }),
    ];

    const groups = groupBySection(grades);
    expect(groups).toHaveLength(1);
    expect(groups[0].sectionEnrollmentId).toBe("se-1");
    expect(groups[0].evaluations).toHaveLength(2);
  });

  it("S-F2a: section header fields come from first row of the group", () => {
    const grade = makeGrade({
      sectionFinalGrade: "6.0",
      sectionStatus: "passed",
      academicPeriodYear: 2026,
      academicPeriodTerm: 1,
      courseCode: "MAT101",
      courseName: "Matemáticas",
    });

    const [group] = groupBySection([grade]);
    expect(group.courseCode).toBe("MAT101");
    expect(group.courseName).toBe("Matemáticas");
    expect(group.period).toBe("2026-1");
    expect(group.finalGrade).toBe("6.0");
    expect(group.status).toBe("Aprobado");
  });

  it("S-F2b: empty sectionFinalGrade passes through as empty string (UI renders —)", () => {
    const grade = makeGrade({
      sectionFinalGrade: "",
      sectionStatus: "in_progress",
    });
    const [group] = groupBySection([grade]);
    // groupBySection does NOT substitute "—" — that is the UI's responsibility.
    expect(group.finalGrade).toBe("");
    expect(group.status).toBe("En curso");
  });

  it("S-F2c: evaluations are sorted ascending by position", () => {
    const grades = [
      makeGrade({
        id: "g1",
        evaluationId: "e1",
        evaluationPosition: 3,
        value: "5.0",
      }),
      makeGrade({
        id: "g2",
        evaluationId: "e2",
        evaluationPosition: 1,
        value: "6.0",
      }),
      makeGrade({
        id: "g3",
        evaluationId: "e3",
        evaluationPosition: 2,
        value: "5.5",
      }),
    ];

    const [group] = groupBySection(grades);
    expect(group.evaluations.map((e) => e.position)).toEqual([1, 2, 3]);
  });

  it("preserves insertion order across distinct sections", () => {
    const gradesSeA = [
      makeGrade({
        id: "g1",
        sectionEnrollmentId: "se-a",
        evaluationPosition: 1,
      }),
    ];
    const gradesSeB = [
      makeGrade({
        id: "g2",
        sectionEnrollmentId: "se-b",
        evaluationPosition: 1,
        courseCode: "FIS201",
      }),
    ];

    const groups = groupBySection([...gradesSeA, ...gradesSeB]);
    expect(groups).toHaveLength(2);
    expect(groups[0].sectionEnrollmentId).toBe("se-a");
    expect(groups[1].sectionEnrollmentId).toBe("se-b");
  });

  it("returns empty array for empty input", () => {
    expect(groupBySection([])).toEqual([]);
  });

  it("S-F3a: evaluation rows carry evaluationId, position, weight, and value — no name field", () => {
    const grade = makeGrade({
      evaluationId: "eval-42",
      evaluationPosition: 2,
      evaluationWeight: "0.400",
      value: "5.5",
    });
    const [group] = groupBySection([grade]);
    const ev = group.evaluations[0];
    expect(ev.evaluationId).toBe("eval-42");
    expect(ev.position).toBe(2);
    expect(ev.weight).toBe("0.400");
    expect(ev.value).toBe("5.5");
    // Evaluation rows must not have a name field.
    expect("name" in ev).toBe(false);
  });

  it("S-F3b: empty value passes through as empty string (UI renders —)", () => {
    const grade = makeGrade({ evaluationId: "e-pending", value: "" });
    const [group] = groupBySection([grade]);
    // groupBySection does NOT substitute "—" — that is the UI's responsibility.
    expect(group.evaluations[0].value).toBe("");
  });
});
