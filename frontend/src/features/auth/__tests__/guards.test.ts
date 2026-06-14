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
      requireRoutePermission(session(["users.manage"]), "/users"),
    ).not.toThrow();
  });

  it("redirects to /forbidden when the session lacks the route's permission", () => {
    let thrown: unknown;
    try {
      requireRoutePermission(session([]), "/users");
    } catch (error) {
      thrown = error;
    }
    expect(thrown).toMatchObject({ options: { to: "/forbidden" } });
  });
});
