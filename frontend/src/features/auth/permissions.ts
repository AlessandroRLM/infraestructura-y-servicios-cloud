// Mirrors the RBAC seeds in backend/migrations (000004, 000006, 000012) —
// the source of truth. Update when a migration adds codes.
export const ROLES = ["admin", "teacher", "student"] as const;
export type Role = (typeof ROLES)[number];

export const PERMISSIONS = [
  "audit.read",
  "catalog.manage",
  "enrollment.manage",
  "enrollment.view_own",
  "grades.override",
  "grades.read",
  "grades.view_own",
  "grades.write",
  "profile.view_own",
  "reports.read",
  "section_enrollment.view_own",
  "sections.enroll",
  "users.manage",
] as const;
export type Permission = (typeof PERMISSIONS)[number];
