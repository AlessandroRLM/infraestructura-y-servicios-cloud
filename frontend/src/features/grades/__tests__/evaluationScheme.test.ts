import { describe, expect, it } from "vitest";
import { evaluationSchemeSchema } from "../schemas/evaluationScheme";

// Direct safeParse assertions for the evaluationSchemeSchema.
// These tests make the weights.ts reference to "zod schema tests" concrete.

describe("evaluationSchemeSchema — row-level validation", () => {
  it("rejects a decimal percent value (.int() guard)", () => {
    const result = evaluationSchemeSchema.safeParse({
      rows: [{ percent: 33.5 }],
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      const msgs = result.error.issues.map((i) => i.message);
      expect(msgs.some((m) => /enteros/i.test(m))).toBe(true);
    }
  });

  it("rejects a percent value below the minimum (.min(1))", () => {
    const result = evaluationSchemeSchema.safeParse({
      rows: [{ percent: 0 }],
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      const msgs = result.error.issues.map((i) => i.message);
      expect(msgs.some((m) => /mínimo/i.test(m))).toBe(true);
    }
  });

  it("rejects a percent value above the maximum (.max(100))", () => {
    const result = evaluationSchemeSchema.safeParse({
      rows: [{ percent: 101 }],
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      const msgs = result.error.issues.map((i) => i.message);
      expect(msgs.some((m) => /máximo/i.test(m))).toBe(true);
    }
  });
});

describe("evaluationSchemeSchema — coercion from string input", () => {
  it("coerces a string percent to number and accepts when total is 100", () => {
    const result = evaluationSchemeSchema.safeParse({
      rows: [{ percent: "100" }],
    });
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.rows[0].percent).toBe(100);
    }
  });

  it("rejects a non-numeric string with the Ingresa un porcentaje message", () => {
    const result = evaluationSchemeSchema.safeParse({
      rows: [{ percent: "abc" }],
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      const msgs = result.error.issues.map((i) => i.message);
      expect(msgs.some((m) => /ingresa un porcentaje/i.test(m))).toBe(true);
    }
  });
});

describe("evaluationSchemeSchema — rows array validation", () => {
  it("rejects an empty rows array (.min(1))", () => {
    const result = evaluationSchemeSchema.safeParse({ rows: [] });
    expect(result.success).toBe(false);
    if (!result.success) {
      const msgs = result.error.issues.map((i) => i.message);
      expect(msgs.some((m) => /al menos una/i.test(m))).toBe(true);
    }
  });
});

describe("evaluationSchemeSchema — cross-field refine (total === 100)", () => {
  it("accepts rows that sum to exactly 100", () => {
    const result = evaluationSchemeSchema.safeParse({
      rows: [{ percent: 30 }, { percent: 30 }, { percent: 40 }],
    });
    expect(result.success).toBe(true);
  });

  it("rejects rows that sum to 99", () => {
    const result = evaluationSchemeSchema.safeParse({
      rows: [{ percent: 50 }, { percent: 49 }],
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      const msgs = result.error.issues.map((i) => i.message);
      expect(msgs.some((m) => /100/i.test(m))).toBe(true);
    }
  });

  it("rejects rows that sum to 101", () => {
    const result = evaluationSchemeSchema.safeParse({
      rows: [{ percent: 51 }, { percent: 50 }],
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      const msgs = result.error.issues.map((i) => i.message);
      expect(msgs.some((m) => /100/i.test(m))).toBe(true);
    }
  });
});
