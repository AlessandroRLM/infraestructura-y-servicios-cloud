import { describe, expect, it } from "vitest";
import {
  ADMIN_PERMISSIONS,
  eligibleAreas,
  isEligibleFor,
  PARTICIPANT_PERMISSIONS,
} from "../area";
import type { AuthenticatedSession } from "../types";

function session(permissions: string[]): AuthenticatedSession {
  return { userId: "1", email: "test@test.com", roles: [], permissions };
}

/** Full teacher permission set as granted by the backend. */
const TEACHER_PERMISSIONS = [
  "grades.read",
  "grades.write",
  "reports.read",
  "section.view_teaching",
  "section_enrollment.view_teaching",
  "profile.view_own",
  "profile.edit_own",
  "profile.view_names",
] as const;

describe("isEligibleFor — admin area", () => {
  it("returns true when the session holds a permission in ADMIN_PERMISSIONS", () => {
    expect(isEligibleFor(session(["catalog.manage"]), "admin")).toBe(true);
  });

  it("returns true for every permission in ADMIN_PERMISSIONS", () => {
    for (const perm of ADMIN_PERMISSIONS) {
      expect(isEligibleFor(session([perm]), "admin")).toBe(true);
    }
  });

  it("returns false when the session holds no admin permission", () => {
    expect(isEligibleFor(session(["grades.view_own"]), "admin")).toBe(false);
  });
});

describe("isEligibleFor — participant area", () => {
  it("returns true when the session holds a permission in PARTICIPANT_PERMISSIONS", () => {
    expect(isEligibleFor(session(["grades.view_own"]), "participant")).toBe(
      true,
    );
  });

  it("returns true for every permission in PARTICIPANT_PERMISSIONS", () => {
    for (const perm of PARTICIPANT_PERMISSIONS) {
      expect(isEligibleFor(session([perm]), "participant")).toBe(true);
    }
  });

  it("returns false when the session holds no participant permission", () => {
    expect(isEligibleFor(session(["catalog.manage"]), "participant")).toBe(
      false,
    );
  });
});

describe("isEligibleFor — enrollment.view_own qualifies for participant area (WU5)", () => {
  it("a session with only enrollment.view_own is eligible for the participant area", () => {
    expect(isEligibleFor(session(["enrollment.view_own"]), "participant")).toBe(
      true,
    );
  });

  it("a session with only enrollment.view_own is NOT eligible for the admin area", () => {
    expect(isEligibleFor(session(["enrollment.view_own"]), "admin")).toBe(
      false,
    );
  });
});

describe("isEligibleFor — grades.write is admin-only (no longer dual-eligible)", () => {
  it("grades.write qualifies for admin", () => {
    const s = session(["grades.write"]);
    expect(isEligibleFor(s, "admin")).toBe(true);
  });

  it("grades.write does NOT qualify for participant", () => {
    const s = session(["grades.write"]);
    expect(isEligibleFor(s, "participant")).toBe(false);
  });
});

describe("eligibleAreas", () => {
  it("S-01a — admin-only user: returns [admin]", () => {
    expect(eligibleAreas(session(["catalog.manage"]))).toEqual(["admin"]);
  });

  it("S-01b — student user: returns [participant]", () => {
    expect(
      eligibleAreas(
        session(["grades.view_own", "section_enrollment.view_own"]),
      ),
    ).toEqual(["participant"]);
  });

  it("S-01c — teacher (full permission set): returns [admin] only", () => {
    expect(eligibleAreas(session([...TEACHER_PERMISSIONS]))).toEqual(["admin"]);
  });

  it("S-01c-simple — teacher (grades.write only): returns [admin] only", () => {
    expect(eligibleAreas(session(["grades.write"]))).toEqual(["admin"]);
  });

  it("S-01d — zero-eligibility user: returns []", () => {
    expect(eligibleAreas(session([]))).toEqual([]);
  });

  it("S-01e — admin with all perms remains dual-eligible for admin and participant", () => {
    // An admin holds both admin perms (catalog.manage) and participant perms
    // (grades.view_own), so they remain dual-eligible for area switching.
    expect(
      eligibleAreas(session(["catalog.manage", "grades.view_own"])),
    ).toEqual(["admin", "participant"]);
  });
});
