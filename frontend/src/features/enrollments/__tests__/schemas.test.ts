import { describe, expect, it } from "vitest";
import {
  adminEnrollmentsSearchSchema,
  ownEnrollmentsSearchSchema,
} from "../schemas/search";

describe("adminEnrollmentsSearchSchema", () => {
  it("parses valid params correctly", () => {
    const result = adminEnrollmentsSearchSchema.parse({
      q: "García",
      status: "pending",
      year: "2025",
      pageSize: "50",
    });
    expect(result).toEqual({
      q: "García",
      status: "pending",
      year: 2025,
      pageSize: 50,
    });
  });

  it("q defaults to empty string when absent", () => {
    const result = adminEnrollmentsSearchSchema.parse({});
    expect(result.q).toBe("");
  });

  it("out-of-range year coerced to undefined", () => {
    const result = adminEnrollmentsSearchSchema.parse({ year: "99" });
    expect(result.year).toBeUndefined();
  });

  it("year below 2000 coerced to undefined", () => {
    const result = adminEnrollmentsSearchSchema.parse({ year: "1999" });
    expect(result.year).toBeUndefined();
  });

  it("year above 2100 coerced to undefined", () => {
    const result = adminEnrollmentsSearchSchema.parse({ year: "2101" });
    expect(result.year).toBeUndefined();
  });

  it("invalid status coerced to undefined", () => {
    const result = adminEnrollmentsSearchSchema.parse({ status: "unknown" });
    expect(result.status).toBeUndefined();
  });

  it("valid status values parse correctly", () => {
    expect(
      adminEnrollmentsSearchSchema.parse({ status: "pending" }).status,
    ).toBe("pending");
    expect(adminEnrollmentsSearchSchema.parse({ status: "paid" }).status).toBe(
      "paid",
    );
    expect(
      adminEnrollmentsSearchSchema.parse({ status: "cancelled" }).status,
    ).toBe("cancelled");
  });

  it("unknown keys are stripped (studentId, programId)", () => {
    const result = adminEnrollmentsSearchSchema.parse({
      studentId: "some-uuid",
      programId: "another-uuid",
    });
    expect(result).not.toHaveProperty("studentId");
    expect(result).not.toHaveProperty("programId");
  });

  it("pageSize defaults to 20 for invalid value", () => {
    const result = adminEnrollmentsSearchSchema.parse({ pageSize: "999" });
    expect(result.pageSize).toBe(20);
  });

  it("pageSize accepts 20, 50, 100", () => {
    expect(
      adminEnrollmentsSearchSchema.parse({ pageSize: "20" }).pageSize,
    ).toBe(20);
    expect(
      adminEnrollmentsSearchSchema.parse({ pageSize: "50" }).pageSize,
    ).toBe(50);
    expect(
      adminEnrollmentsSearchSchema.parse({ pageSize: "100" }).pageSize,
    ).toBe(100);
  });

  it("invalid q value falls back to empty string", () => {
    // .catch("") means any parse failure returns ""
    const result = adminEnrollmentsSearchSchema.parse({ q: undefined });
    expect(result.q).toBe("");
  });
});

describe("ownEnrollmentsSearchSchema", () => {
  it("parses pageSize only", () => {
    const result = ownEnrollmentsSearchSchema.parse({ pageSize: "50" });
    expect(result).toEqual({ pageSize: 50 });
  });

  it("status is stripped as unknown key", () => {
    const result = ownEnrollmentsSearchSchema.parse({
      status: "paid",
      year: "2025",
    });
    expect(result).not.toHaveProperty("status");
    expect(result).not.toHaveProperty("year");
  });

  it("pageSize defaults to 20 for invalid value", () => {
    const result = ownEnrollmentsSearchSchema.parse({});
    expect(result.pageSize).toBe(20);
  });
});
