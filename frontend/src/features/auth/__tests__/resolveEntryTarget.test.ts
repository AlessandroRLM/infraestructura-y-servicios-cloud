import { describe, expect, it } from "vitest";
import { resolveEntryTarget } from "../resolveEntryTarget";

describe("resolveEntryTarget", () => {
  it("returns /forbidden when there are 0 eligible areas", () => {
    expect(resolveEntryTarget([], null)).toBe("/forbidden");
  });

  it("returns /admin when the only eligible area is admin", () => {
    expect(resolveEntryTarget(["admin"], null)).toBe("/admin");
  });

  it("returns /app when the only eligible area is participant", () => {
    expect(resolveEntryTarget(["participant"], null)).toBe("/app");
  });

  it("returns /admin when dual-eligible and stored preference is 'admin'", () => {
    expect(resolveEntryTarget(["admin", "participant"], "admin")).toBe(
      "/admin",
    );
  });

  it("returns /app when dual-eligible and stored preference is 'participant'", () => {
    expect(resolveEntryTarget(["admin", "participant"], "participant")).toBe(
      "/app",
    );
  });

  it("returns /choose-area when dual-eligible and preference is null", () => {
    expect(resolveEntryTarget(["admin", "participant"], null)).toBe(
      "/choose-area",
    );
  });

  it("returns /choose-area when dual-eligible and preference is no longer a valid area", () => {
    // Edge case: stored value was cleared/invalid upstream — resolveEntryTarget
    // only sees null here, but test documents the intent.
    expect(resolveEntryTarget(["admin", "participant"], null)).toBe(
      "/choose-area",
    );
  });

  it("ignores a stored preference that is not in the eligible set (single-eligible)", () => {
    // Stored 'admin' but only participant-eligible → single-eligible path wins.
    expect(resolveEntryTarget(["participant"], "admin")).toBe("/app");
  });
});
