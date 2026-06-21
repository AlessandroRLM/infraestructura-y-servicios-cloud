import { describe, expect, it } from "vitest";
import type { SessionState } from "@/features/auth";
import {
  ADMIN_NAV,
  firstAccessibleNavTarget,
  PARTICIPANT_NAV,
} from "../AppSidebar";

function authenticatedSession(permissions: string[]): SessionState {
  return {
    status: "authenticated",
    userId: "1",
    email: "user@test.com",
    roles: [],
    permissions,
  };
}

describe("firstAccessibleNavTarget — admin nav", () => {
  it("returns academics target for a full-admin session (catalog.manage)", () => {
    const session = authenticatedSession(["catalog.manage"]);
    const target = firstAccessibleNavTarget(session, ADMIN_NAV);
    expect(target).toBeDefined();
    expect(target?.to).toBe("/admin/academics");
  });

  it("returns grades target for a teacher session (grades.write only)", () => {
    const session = authenticatedSession(["grades.write"]);
    const target = firstAccessibleNavTarget(session, ADMIN_NAV);
    expect(target).toBeDefined();
    expect(target?.to).toBe("/admin/grades");
  });

  it("returns reports target for a session with only reports.read", () => {
    const session = authenticatedSession(["reports.read"]);
    const target = firstAccessibleNavTarget(session, ADMIN_NAV);
    expect(target).toBeDefined();
    expect(target?.to).toBe("/admin/reports");
  });

  it("returns undefined when no admin nav item is accessible", () => {
    const session = authenticatedSession(["grades.view_own"]);
    const target = firstAccessibleNavTarget(session, ADMIN_NAV);
    expect(target).toBeUndefined();
  });
});

describe("firstAccessibleNavTarget — participant nav", () => {
  it("returns grades target for a student session (grades.view_own)", () => {
    const session = authenticatedSession(["grades.view_own"]);
    const target = firstAccessibleNavTarget(session, PARTICIPANT_NAV);
    expect(target).toBeDefined();
    expect(target?.to).toBe("/app/grades");
  });

  it("returns undefined when no participant nav item is accessible", () => {
    const session = authenticatedSession(["catalog.manage"]);
    const target = firstAccessibleNavTarget(session, PARTICIPANT_NAV);
    expect(target).toBeUndefined();
  });
});

describe("firstAccessibleNavTarget — unauthenticated/loading session", () => {
  it("returns undefined for unauthenticated session", () => {
    const session: SessionState = { status: "unauthenticated" };
    const target = firstAccessibleNavTarget(session, ADMIN_NAV);
    expect(target).toBeUndefined();
  });
});
