import type { Role } from "./permissions";

/** Human-readable display label for each role. */
export const ROLE_LABELS: Record<Role, string> = {
  admin: "Administrador",
  teacher: "Profesor",
  student: "Estudiante",
};

/**
 * Roles ordered from most to least privileged.
 * Use this order for display and priority comparisons.
 */
export const ROLE_PRIORITY: Role[] = ["admin", "teacher", "student"];

/** Returns the display label for a single role. */
export function roleLabel(role: Role): string {
  return ROLE_LABELS[role];
}

/**
 * Returns the primary (highest-priority) role label for a user.
 * Falls back to the first role in the list if none matches ROLE_PRIORITY.
 */
export function primaryRoleLabel(roles: string[]): string {
  const primary = ROLE_PRIORITY.find((r) => roles.includes(r)) ?? roles[0];
  return primary ? (ROLE_LABELS[primary as Role] ?? primary) : "";
}

/** Sorts an array of Role values by ROLE_PRIORITY (admin first). */
export function sortRolesByPriority(roles: Role[]): Role[] {
  return [...roles].sort(
    (a, b) => ROLE_PRIORITY.indexOf(a) - ROLE_PRIORITY.indexOf(b),
  );
}
