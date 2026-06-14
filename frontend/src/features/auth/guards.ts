import { redirect } from "@tanstack/react-router";
import { hasPermission } from "./hooks/useSession";
import type { Permission } from "./permissions";
import { type GuardedRoute, ROUTE_PERMISSIONS } from "./routePermissions";
import type { AuthenticatedSession, SessionState } from "./types";

/**
 * Route-level authorization primitive. UX only — the backend enforces
 * authorization per RPC (fail-closed); this just keeps an authenticated user
 * without the permission off a page that would otherwise render broken.
 *
 * ANY semantics: holding one of `permissions` is enough. Call from a route's
 * `beforeLoad`, after the `_authenticated` layout has put the session on the
 * context. Prefer {@link requireRoutePermission} so the permission stays
 * declared in a single place.
 *
 * @throws a redirect to `/forbidden` when the session holds none of `permissions`.
 */
export function requireAnyPermission(
  session: AuthenticatedSession,
  permissions: readonly Permission[],
): void {
  // Reuse the single membership check rather than re-reading session.permissions
  // here, so both the guards and useSession consumers stay in sync.
  const state: SessionState = { status: "authenticated", ...session };
  if (permissions.some((permission) => hasPermission(state, permission))) {
    return;
  }
  throw redirect({ to: "/forbidden" });
}

/**
 * Guards a route using {@link ROUTE_PERMISSIONS} — the single source of truth
 * shared with the nav. Pass the route's URL path; the required permission is
 * resolved from the map, so it is never duplicated in the route file.
 *
 * @throws a redirect to `/forbidden` when the session lacks access to `route`.
 */
export function requireRoutePermission(
  session: AuthenticatedSession,
  route: GuardedRoute,
): void {
  requireAnyPermission(session, ROUTE_PERMISSIONS[route]);
}
