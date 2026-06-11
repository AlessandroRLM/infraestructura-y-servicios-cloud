import { useSessionContext } from "./context";
import type { SessionData } from "./source";

export function useSession(): SessionData {
  return useSessionContext();
}

export function hasPermission(
  session: SessionData,
  permission: string,
): boolean {
  return (
    session.status === "authenticated" &&
    session.permissions.includes(permission)
  );
}

export function hasRole(session: SessionData, role: string): boolean {
  return session.status === "authenticated" && session.roles.includes(role);
}
