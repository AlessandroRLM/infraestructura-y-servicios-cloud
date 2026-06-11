import { describe, expect, it } from "vitest";
import { stubSessionSource } from "../stub";
import { hasPermission, hasRole } from "../useSession";

describe("stubSessionSource", () => {
  it("returns unauthenticated status", async () => {
    const session = await stubSessionSource.getSession();
    expect(session.status).toBe("unauthenticated");
  });

  it("returns null user", async () => {
    const session = await stubSessionSource.getSession();
    expect(session.user).toBeNull();
  });

  it("returns empty roles and permissions", async () => {
    const session = await stubSessionSource.getSession();
    expect(session.roles).toEqual([]);
    expect(session.permissions).toEqual([]);
  });
});

describe("hasPermission()", () => {
  it("returns false for unauthenticated session", async () => {
    const session = await stubSessionSource.getSession();
    expect(hasPermission(session, "grades.write")).toBe(false);
  });

  it("returns true when permission is present", async () => {
    const session = {
      user: { id: "1", email: "a@b.com" },
      roles: ["teacher"],
      permissions: ["grades.write"],
      status: "authenticated" as const,
    };
    expect(hasPermission(session, "grades.write")).toBe(true);
  });
});

describe("hasRole()", () => {
  it("returns false for unauthenticated session", async () => {
    const session = await stubSessionSource.getSession();
    expect(hasRole(session, "teacher")).toBe(false);
  });

  it("returns true when role is present", async () => {
    const session = {
      user: { id: "1", email: "a@b.com" },
      roles: ["teacher"],
      permissions: [],
      status: "authenticated" as const,
    };
    expect(hasRole(session, "teacher")).toBe(true);
  });
});
