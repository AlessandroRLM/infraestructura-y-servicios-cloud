import { hasPermission, useSession } from "@/features/auth";
import { SchemeManagementView } from "./SchemeManagementView";

/**
 * Placeholder rendered at /admin/grades for sessions that lack grades.override.
 * Kept as a named component so the gate below reads clearly.
 */
function SchemePlaceholder() {
  return (
    <div className="space-y-1">
      <h1 className="font-semibold text-2xl tracking-tight">Notas</h1>
      <p className="text-muted-foreground">Registro de notas — próximamente.</p>
    </div>
  );
}

/**
 * Entry point for the admin grades area at /admin/grades.
 *
 * Gates on `grades.override`:
 * - Granted  → renders SchemeManagementView (evaluation-scheme admin UI).
 * - Absent   → renders the SchemePlaceholder ("próximamente").
 *
 * No server call is made before the permission check resolves.
 */
export function GradesPage() {
  const session = useSession();

  if (hasPermission(session, "grades.override")) {
    return <SchemeManagementView />;
  }

  return <SchemePlaceholder />;
}
