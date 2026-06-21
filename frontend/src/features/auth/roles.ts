import type { Role } from "./permissions";

/** Human-readable display label for each role. */
export const ROLE_LABELS: Record<Role, string> = {
  admin: "Administrador",
  teacher: "Profesor",
  student: "Estudiante",
};

/** Returns the display label for a single role. */
export function roleLabel(role: Role): string {
  return ROLE_LABELS[role];
}
