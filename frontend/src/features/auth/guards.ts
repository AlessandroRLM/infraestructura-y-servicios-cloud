import { redirect } from "@tanstack/react-router";
import type { Permission } from "./permissions";
import type { AuthenticatedSession } from "./types";

// Route-level authorization. This is UX, not security — the backend enforces
// authorization per RPC (fail-closed). Its only job is to keep an authenticated
// user without the permission off a page that would otherwise render broken,
// and send them to a 403 instead.
//
// ANY semantics mirror the nav's link-visibility rule: holding one of the
// listed permissions is enough. Call from a route's beforeLoad, after the
// _authenticated layout has put the session on the context.
export function requireAnyPermission(
  session: AuthenticatedSession,
  permissions: readonly Permission[],
): void {
  if (
    permissions.some((permission) => session.permissions.includes(permission))
  ) {
    return;
  }
  throw redirect({ to: "/forbidden" });
}
