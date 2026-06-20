import { describe, expect, it } from "vitest";
import { requireAnyPermission, requireRoutePermission } from "../guards";
import type { AuthenticatedSession } from "../types";

function session(permissions: string[]): AuthenticatedSession {
  return {
    userId: "1",
    email: "user@test.com",
    roles: ["admin"],
    permissions,
  };
}

describe("requireAnyPermission", () => {
  it("returns without throwing when the session holds the required permission", () => {
    expect(() =>
      requireAnyPermission(session(["users.manage"]), ["users.manage"]),
    ).not.toThrow();
  });

  it("passes when the session holds ANY one of several accepted permissions", () => {
    expect(() =>
      requireAnyPermission(session(["section_enrollment.view_own"]), [
        "sections.enroll",
        "section_enrollment.view_own",
      ]),
    ).not.toThrow();
  });

  it("throws a redirect to /forbidden when the session lacks every accepted permission", () => {
    let thrown: unknown;
    try {
      requireAnyPermission(session([]), ["catalog.manage"]);
    } catch (error) {
      thrown = error;
    }
    expect(thrown).toMatchObject({ options: { to: "/forbidden" } });
  });
});

describe("requireRoutePermission", () => {
  it("resolves the route's permission from ROUTE_PERMISSIONS and allows access", () => {
    expect(() =>
      requireRoutePermission(session(["users.manage"]), "/admin/users"),
    ).not.toThrow();
  });

  it("redirects to /forbidden when the session lacks the route's permission", () => {
    let thrown: unknown;
    try {
      requireRoutePermission(session([]), "/admin/users");
    } catch (error) {
      thrown = error;
    }
    expect(thrown).toMatchObject({ options: { to: "/forbidden" } });
  });

  it("/admin/grades requires grades.read or grades.write (admin perspective)", () => {
    expect(() =>
      requireRoutePermission(session(["grades.read"]), "/admin/grades"),
    ).not.toThrow();
    expect(() =>
      requireRoutePermission(session(["grades.write"]), "/admin/grades"),
    ).not.toThrow();
  });

  it("/app/grades requires grades.view_own (participant perspective)", () => {
    expect(() =>
      requireRoutePermission(session(["grades.view_own"]), "/app/grades"),
    ).not.toThrow();
  });

  it("/app/enrollments requires enrollment.view_own (WU5)", () => {
    expect(() =>
      requireRoutePermission(
        session(["enrollment.view_own"]),
        "/app/enrollments",
      ),
    ).not.toThrow();
  });

  it("/app/enrollments rejects a session without enrollment.view_own (WU5)", () => {
    let thrown: unknown;
    try {
      requireRoutePermission(session(["grades.view_own"]), "/app/enrollments");
    } catch (error) {
      thrown = error;
    }
    expect(thrown).toMatchObject({ options: { to: "/forbidden" } });
  });

  it("/admin/grades and /app/grades require distinct permission sets", () => {
    // grades.view_own is NOT sufficient for the admin grades route
    let thrown: unknown;
    try {
      requireRoutePermission(session(["grades.view_own"]), "/admin/grades");
    } catch (error) {
      thrown = error;
    }
    expect(thrown).toMatchObject({ options: { to: "/forbidden" } });
  });
});
