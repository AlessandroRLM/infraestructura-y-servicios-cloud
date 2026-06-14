import type { Permission } from "./permissions";

/**
 * Single source of truth for which permission(s) gate each protected route.
 * Consumed by both the route `beforeLoad` guards (via `requireRoutePermission`)
 * and the nav's link visibility (`AppSidebar`), so a route's authorization is
 * declared once instead of duplicated across the two.
 *
 * ANY semantics: holding one of the listed permissions is enough. Routes absent
 * from this map (`/`, `/profile`, `/forbidden`) require no permission.
 *
 * This is UX only — the backend enforces authorization per RPC (fail-closed).
 */
export const ROUTE_PERMISSIONS = {
  "/academics": ["catalog.manage"],
  "/enrollments": ["enrollment.manage"],
  "/section-enrollments": ["sections.enroll", "section_enrollment.view_own"],
  "/grades": ["grades.read", "grades.write", "grades.view_own"],
  "/reports": ["reports.read"],
  "/users": ["users.manage"],
  "/access-control": ["users.manage"],
} as const satisfies Record<string, readonly Permission[]>;

export type GuardedRoute = keyof typeof ROUTE_PERMISSIONS;

/**
 * Looks up the permissions guarding `path`. Accepts any path — e.g. a nav
 * link's `to`, which spans every route, not just guarded ones — and returns
 * `undefined` when the route is unguarded.
 */
export function routePermissions(
  path: string,
): readonly Permission[] | undefined {
  return (ROUTE_PERMISSIONS as Record<string, readonly Permission[]>)[path];
}
