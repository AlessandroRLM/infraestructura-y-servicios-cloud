import { describe, expect, it } from "vitest";
import { gradeValueSchema } from "../gradeValue";

describe("gradeValueSchema", () => {
  it("accepts 1.0 (minimum)", () => {
    expect(gradeValueSchema.safeParse("1.0").success).toBe(true);
    expect(gradeValueSchema.safeParse(1.0).success).toBe(true);
  });

  it("accepts 7.0 (maximum)", () => {
    expect(gradeValueSchema.safeParse("7.0").success).toBe(true);
    expect(gradeValueSchema.safeParse(7.0).success).toBe(true);
  });

  it("accepts 6.5 (valid one decimal)", () => {
    expect(gradeValueSchema.safeParse("6.5").success).toBe(true);
  });

  it("accepts 4.0 (whole number)", () => {
    expect(gradeValueSchema.safeParse("4.0").success).toBe(true);
    expect(gradeValueSchema.safeParse("4").success).toBe(true);
  });

  it("rejects 0.9 (below minimum)", () => {
    const result = gradeValueSchema.safeParse("0.9");
    expect(result.success).toBe(false);
  });

  it("rejects 7.1 (above maximum)", () => {
    const result = gradeValueSchema.safeParse("7.1");
    expect(result.success).toBe(false);
  });

  it("rejects 8.5 (well above maximum)", () => {
    const result = gradeValueSchema.safeParse("8.5");
    expect(result.success).toBe(false);
  });

  it("rejects 5.55 (two decimal places)", () => {
    const result = gradeValueSchema.safeParse("5.55");
    expect(result.success).toBe(false);
  });

  it("rejects 1.23 (two decimal places)", () => {
    const result = gradeValueSchema.safeParse("1.23");
    expect(result.success).toBe(false);
  });

  it("rejects non-numeric string", () => {
    const result = gradeValueSchema.safeParse("abc");
    expect(result.success).toBe(false);
  });

  it("rejects empty string", () => {
    const result = gradeValueSchema.safeParse("");
    expect(result.success).toBe(false);
  });

  it("provides a user-facing error message for out-of-range", () => {
    const result = gradeValueSchema.safeParse("8.0");
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues[0]?.message).toBeTruthy();
    }
  });
});
