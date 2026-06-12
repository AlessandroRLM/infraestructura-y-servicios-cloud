import { useContext } from "react";
import { SessionContext } from "../context/context";
import type { Permission, Role } from "../permissions";
import type { SessionState } from "../types";

export function useSession(): SessionState {
  const session = useContext(SessionContext);
  if (session === null) {
    throw new Error("useSession must be used within a SessionProvider");
  }
  return session;
}

// `permission`/`role` are typed unions so call sites fail at compile time on
// typos; the session arrays stay string[] because that is what the wire
// carries — the backend may know codes this frontend does not yet.
export function hasPermission(
  session: SessionState,
  permission: Permission,
): boolean {
  return (
    session.status === "authenticated" &&
    session.permissions.includes(permission)
  );
}

export function hasRole(session: SessionState, role: Role): boolean {
  return session.status === "authenticated" && session.roles.includes(role);
}
