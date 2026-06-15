import type { AuthenticatedSession } from "./types";

/** The two top-level authenticated areas the application exposes. */
export type Area = "admin" | "participant";

/**
 * Permissions that qualify a user for the admin area.
 * `grades.write` is included because teachers manage grades (dual-eligible overlap).
 */
export const ADMIN_PERMISSIONS = [
  "catalog.manage",
  "users.manage",
  "enrollment.manage",
  "grades.read",
  "grades.write",
  "reports.read",
  "audit.read",
] as const;

/**
 * Permissions that qualify a user for the participant area.
 * `grades.write` is included because teachers can also enter their own context (dual-eligible overlap).
 */
export const PARTICIPANT_PERMISSIONS = [
  "grades.view_own",
  "section_enrollment.view_own",
  "sections.enroll",
  "grades.write",
  "profile.view_own",
  "profile.edit_own",
] as const;

/**
 * Returns `true` when the session holds at least one permission from the
 * given area's permission set. Pure function — no router or DOM deps.
 */
export function isEligibleFor(
  session: AuthenticatedSession,
  area: Area,
): boolean {
  const set =
    area === "admin" ? ADMIN_PERMISSIONS : PARTICIPANT_PERMISSIONS;
  return set.some((p) => session.permissions.includes(p));
}

/**
 * Returns the ordered list of areas the session is eligible for.
 * The result is `[]` for a zero-eligibility session, `["admin"]` or
 * `["participant"]` for single-eligible, and `["admin","participant"]`
 * for dual-eligible (e.g. a teacher with `grades.write`).
 */
export function eligibleAreas(session: AuthenticatedSession): Area[] {
  const areas: Area[] = [];
  if (isEligibleFor(session, "admin")) areas.push("admin");
  if (isEligibleFor(session, "participant")) areas.push("participant");
  return areas;
}
