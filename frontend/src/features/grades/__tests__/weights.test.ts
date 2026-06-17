import { describe, expect, it } from "vitest";
import { percentToWeight, sumPercents, weightToPercent } from "../weights";

// NOTE: percentToWeight trusts its caller — it does NOT validate that the
// input is an integer. That invariant is enforced by the zod schema
// (z.coerce.number().int()) in evaluationScheme.ts BEFORE conversion.
// The tests below document the happy-path contract (integer inputs only).
// Non-integer protection is validated by the zod schema tests.

describe("percentToWeight", () => {
  it("converts 30 to '0.300'", () => {
    expect(percentToWeight(30)).toBe("0.300");
  });

  it("converts 40 to '0.400'", () => {
    expect(percentToWeight(40)).toBe("0.400");
  });

  it("converts 5 to '0.050'", () => {
    expect(percentToWeight(5)).toBe("0.050");
  });

  it("converts 100 to '1.000'", () => {
    expect(percentToWeight(100)).toBe("1.000");
  });

  it("converts 1 to '0.010'", () => {
    expect(percentToWeight(1)).toBe("0.010");
  });

  it("maps [30, 30, 40] to ['0.300', '0.300', '0.400']", () => {
    expect([30, 30, 40].map(percentToWeight)).toEqual([
      "0.300",
      "0.300",
      "0.400",
    ]);
  });
});

describe("weightToPercent", () => {
  it("converts '0.400' to 40", () => {
    expect(weightToPercent("0.400")).toBe(40);
  });

  it("converts '0.050' to 5", () => {
    expect(weightToPercent("0.050")).toBe(5);
  });

  it("converts '1.000' to 100", () => {
    expect(weightToPercent("1.000")).toBe(100);
  });

  it("converts '0.300' to 30", () => {
    expect(weightToPercent("0.300")).toBe(30);
  });
});

describe("sumPercents", () => {
  it("sums [30, 30, 40] to 100", () => {
    expect(
      sumPercents([{ percent: 30 }, { percent: 30 }, { percent: 40 }]),
    ).toBe(100);
  });

  it("sums a single 100% row to 100", () => {
    expect(sumPercents([{ percent: 100 }])).toBe(100);
  });

  it("returns 0 for an empty array", () => {
    expect(sumPercents([])).toBe(0);
  });

  it("returns partial sum when total is not 100", () => {
    expect(sumPercents([{ percent: 30 }, { percent: 30 }])).toBe(60);
  });
});
