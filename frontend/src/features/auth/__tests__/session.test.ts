import { describe, expect, it } from "vitest";
import { stubSessionSource } from "../api/stub";
import { hasPermission, hasRole } from "../hooks/useSession";
import type { SessionState } from "../types";

const authenticated: SessionState = {
  status: "authenticated",
  userId: "1",
  email: "a@b.com",
  roles: ["teacher"],
  permissions: ["grades.write"],
};

describe("stubSessionSource", () => {
  it("resolves to null (no active session)", async () => {
    await expect(stubSessionSource.getSession()).resolves.toBeNull();
  });
});

describe("hasPermission()", () => {
  it("returns false while loading", () => {
    expect(hasPermission({ status: "loading" }, "grades.write")).toBe(false);
  });

  it("returns false for unauthenticated session", () => {
    expect(hasPermission({ status: "unauthenticated" }, "grades.write")).toBe(
      false,
    );
  });

  it("returns false on error", () => {
    expect(hasPermission({ status: "error" }, "grades.write")).toBe(false);
  });

  it("returns true when the permission is present", () => {
    expect(hasPermission(authenticated, "grades.write")).toBe(true);
  });

  it("returns false when the permission is absent", () => {
    expect(hasPermission(authenticated, "catalog.manage")).toBe(false);
  });
});

describe("hasRole()", () => {
  it("returns false while loading", () => {
    expect(hasRole({ status: "loading" }, "teacher")).toBe(false);
  });

  it("returns false for unauthenticated session", () => {
    expect(hasRole({ status: "unauthenticated" }, "teacher")).toBe(false);
  });

  it("returns false on error", () => {
    expect(hasRole({ status: "error" }, "teacher")).toBe(false);
  });

  it("returns true when the role is present", () => {
    expect(hasRole(authenticated, "teacher")).toBe(true);
  });

  it("returns false when the role is absent", () => {
    expect(hasRole(authenticated, "admin")).toBe(false);
  });
});
