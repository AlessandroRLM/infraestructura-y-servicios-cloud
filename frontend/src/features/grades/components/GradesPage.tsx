import { hasPermission, useSession } from "@/features/auth";
import { OwnGradesView } from "./OwnGradesView";

/**
 * Entry-point for the /grades route. Branches on the session permission:
 * - `grades.view_own` → renders the student "Mis notas" accordion view.
 * - Anything else → renders a placeholder for roles that lack own-grade access.
 */
export function GradesPage() {
  const session = useSession();
  const canViewOwn = hasPermission(session, "grades.view_own");

  if (canViewOwn) {
    // OwnGradesView includes its own "Mis notas" heading.
    return <OwnGradesView />;
  }

  return (
    <div className="space-y-1">
      <h1 className="font-semibold text-2xl tracking-tight">Notas</h1>
      <p className="text-muted-foreground">Registro de notas — próximamente.</p>
    </div>
  );
}
