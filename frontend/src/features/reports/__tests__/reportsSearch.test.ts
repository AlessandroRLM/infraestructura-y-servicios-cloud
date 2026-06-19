/**
 * reportsSearch schema unit tests — AC-2.a, AC-2.b, AC-2.c, AC-2.d
 */
import { describe, expect, it } from "vitest";
import { validateSearch } from "../schemas/reportsSearch";

describe("validateSearch — reportsSearch schema", () => {
  it("AC-2.a: valid tab + periodId → correct shape with all defaults", () => {
    const result = validateSearch({ tab: "occupancy", periodId: "p1" });
    expect(result).toEqual({
      tab: "occupancy",
      periodId: "p1",
      sectionId: "",
      programId: "",
      studentId: "",
      year: undefined,
    });
  });

  it("AC-2.b: invalid tab → falls back to section-grade", () => {
    const result = validateSearch({ tab: "bad-value" });
    expect(result.tab).toBe("section-grade");
  });

  it("AC-2.c: year as string '2024' → coerced to number 2024", () => {
    const result = validateSearch({ year: "2024" });
    expect(result.year).toBe(2024);
  });

  it("AC-2.d: year as non-numeric string → returns undefined", () => {
    const result = validateSearch({ year: "abc" });
    expect(result.year).toBeUndefined();
  });

  it("empty input → all defaults", () => {
    const result = validateSearch({});
    expect(result).toEqual({
      tab: "section-grade",
      sectionId: "",
      periodId: "",
      programId: "",
      studentId: "",
      year: undefined,
    });
  });

  it("tab: undefined falls back to section-grade via .default()", () => {
    // In Zod 4 an undefined input is intercepted by the field's .default() (not .catch()),
    // so the tab resolves to the default value.
    const result = validateSearch({ tab: undefined });
    expect(result.tab).toBe("section-grade");
  });

  it("all 4 tab values are valid", () => {
    for (const tab of [
      "section-grade",
      "occupancy",
      "program-summary",
      "student-record",
    ] as const) {
      expect(validateSearch({ tab }).tab).toBe(tab);
    }
  });

  it("year out of range (1999) → returns undefined via catch", () => {
    const result = validateSearch({ year: 1999 });
    expect(result.year).toBeUndefined();
  });

  it("year out of range (2101) → returns undefined via catch", () => {
    const result = validateSearch({ year: 2101 });
    expect(result.year).toBeUndefined();
  });

  it("sectionId provided → preserved", () => {
    const result = validateSearch({ sectionId: "sec-123" });
    expect(result.sectionId).toBe("sec-123");
  });

  // Defensive parse: non-object inputs must NOT throw and must return all defaults.
  const allDefaults = {
    tab: "section-grade",
    sectionId: "",
    periodId: "",
    programId: "",
    studentId: "",
    year: undefined,
  };

  it("null input → all defaults (no throw)", () => {
    expect(validateSearch(null)).toEqual(allDefaults);
  });

  it("number input (42) → all defaults (no throw)", () => {
    expect(validateSearch(42)).toEqual(allDefaults);
  });

  it("string input ('x') → all defaults (no throw)", () => {
    expect(validateSearch("x")).toEqual(allDefaults);
  });
});
