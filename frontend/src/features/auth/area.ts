import type { AuthenticatedSession } from "./types";

/** The two top-level authenticated areas the application exposes. */
export type Area = "admin" | "participant";

/** Permissions that qualify a user for the admin area. */
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
 * Only route-gating permissions are included (see ROUTE_PERMISSIONS).
 * Generic auth perms (profile.*) are intentionally excluded: every
 * authenticated user may hold them, so they cannot be a meaningful
 * eligibility signal. Teachers hold admin perms only and are therefore
 * admin-only eligible, bypassing the /choose-area prompt.
 */
export const PARTICIPANT_PERMISSIONS = [
  "grades.view_own",
  "enrollment.view_own",
  "section_enrollment.view_own",
  "sections.enroll",
] as const;

/**
 * Returns `true` when the session holds at least one permission from the
 * given area's permission set. Pure function — no router or DOM deps.
 */
export function isEligibleFor(
  session: AuthenticatedSession,
  area: Area,
): boolean {
  const set = area === "admin" ? ADMIN_PERMISSIONS : PARTICIPANT_PERMISSIONS;
  return set.some((p) => session.permissions.includes(p));
}

/**
 * Returns the ordered list of areas the session is eligible for.
 * The result is `[]` for a zero-eligibility session, `["admin"]` or
 * `["participant"]` for single-eligible, and `["admin","participant"]`
 * for dual-eligible (e.g. an admin who also holds participant permissions).
 */
export function eligibleAreas(session: AuthenticatedSession): Area[] {
  const areas: Area[] = [];
  if (isEligibleFor(session, "admin")) areas.push("admin");
  if (isEligibleFor(session, "participant")) areas.push("participant");
  return areas;
}
