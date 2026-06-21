import type { Permission } from "./permissions";

/**
 * Single source of truth for which permission(s) gate each protected route.
 * Consumed by both the route `beforeLoad` guards (via `requireRoutePermission`)
 * and the nav's link visibility (`AppSidebar`), so a route's authorization is
 * declared once instead of duplicated across the two.
 *
 * ANY semantics: holding one of the listed permissions is enough. Routes absent
 * from this map (`/`, `/profile`, `/forbidden`, `/choose-area`) require no
 * permission beyond being authenticated.
 *
 * This is UX only — the backend enforces authorization per RPC (fail-closed).
 *
 * Keys use the full prefixed URL path (e.g. `/admin/academics`, `/app/grades`)
 * so the guard key matches the route URL, making it unambiguous which area a
 * feature belongs to.
 *
 * The value type is a non-empty tuple: an empty list would make
 * `requireAnyPermission` reject every session, locking all users out of the
 * route. A guarded route must require at least one permission.
 */
export const ROUTE_PERMISSIONS = {
  "/admin/academics": ["catalog.manage"],
  "/admin/audit": ["audit.read"],
  "/admin/enrollments": ["enrollment.manage"],
  "/admin/section-enrollments": ["enrollment.manage"],
  "/admin/grades": ["grades.read", "grades.write"],
  "/admin/grades/$sectionId": ["grades.read", "grades.write"],
  "/admin/reports": ["reports.read"],
  "/admin/users": ["users.manage"],
  "/app/grades": ["grades.view_own"],
  "/app/enrollments": ["enrollment.view_own"],
  "/app/section-enrollments": [
    "section_enrollment.view_own",
    "sections.enroll",
  ],
} as const satisfies Record<string, readonly [Permission, ...Permission[]]>;

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
