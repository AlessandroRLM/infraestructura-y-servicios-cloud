import { useState } from "react";
import { hasPermission, useSession } from "@/features/auth";
import type { TeachingSection } from "@/gen/catalog/v1/catalog_pb";
import { GradeRecordingGrid } from "./GradeRecordingGrid";
import { SectionSelectionTable } from "./SectionSelectionTable";

/**
 * Entry point for the grades area at /admin/grades.
 *
 * Permission switch:
 * - grades.write (teacher) → recording flow (SectionSelectionTable → GradeRecordingGrid).
 *   No "Administrar Notas" button is shown.
 * - grades.override (admin) → same recording flow + "Administrar Notas" button inside
 *   the section grid (rendered by GradeRecordingGrid when the caller has grades.override).
 * - Neither permission → placeholder.
 *
 * Navigation state is local: selected section lives here so the back button works without
 * a router history entry.
 */
export function GradesPage() {
  const session = useSession();
  const canWrite = hasPermission(session, "grades.write");
  const canOverride = hasPermission(session, "grades.override");

  const [selectedSection, setSelectedSection] =
    useState<TeachingSection | null>(null);

  if (!canWrite && !canOverride) {
    return (
      <div className="flex flex-col gap-1">
        <h1 className="font-semibold text-2xl tracking-tight">Notas</h1>
        <p className="text-muted-foreground">
          No tienes permisos para acceder a esta sección.
        </p>
      </div>
    );
  }

  if (selectedSection) {
    return (
      <GradeRecordingGrid
        section={selectedSection}
        onBack={() => setSelectedSection(null)}
      />
    );
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-1">
        <h1 className="font-semibold text-2xl tracking-tight">Notas</h1>
        <p className="text-muted-foreground text-sm">
          Selecciona una sección para registrar notas.
        </p>
      </div>
      <SectionSelectionTable onSelectSection={setSelectedSection} />
    </div>
  );
}
