import { useNavigate } from "@tanstack/react-router";
import { hasPermission, useSession } from "@/features/auth";
import type { TeachingSection } from "@/gen/catalog/v1/catalog_pb";
import { SectionSelectionTable } from "./SectionSelectionTable";

interface GradesPageProps {
  /** Current search query (URL-synced). */
  q: string;
  /** Current page size (URL-synced). */
  pageSize: number;
  /** Called with the debounced query value; updates the URL search param. */
  onQueryChange: (v: string) => void;
  /** Called when the user picks a different page size; updates the URL search param. */
  onPageSizeChange: (n: number) => void;
}

/**
 * Section-selection view rendered at /admin/grades.
 *
 * Permission switch:
 * - grades.write (teacher) or grades.override (admin) → selection table; clicking a row
 *   navigates to /admin/grades/$sectionId (deep-linkable, refresh-stable).
 * - Neither permission → placeholder.
 *
 * The selected section is NOT held in local state; the sub-route owns it.
 * q and pageSize are owned by the route and passed in as props so this
 * component stays purely presentational with respect to URL state.
 */
export function GradesPage({
  q,
  pageSize,
  onQueryChange,
  onPageSizeChange,
}: GradesPageProps) {
  const session = useSession();
  const canWrite = hasPermission(session, "grades.write");
  const canOverride = hasPermission(session, "grades.override");
  const navigate = useNavigate();

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

  const handleSelectSection = (section: TeachingSection) => {
    // Spread into a plain record so the history state stays serializable.
    // GradesSectionPage re-narrows the state back to TeachingSection.
    navigate({
      to: "/admin/grades/$sectionId",
      params: { sectionId: section.id },
      state: { section: { ...section } },
    });
  };

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-1">
        <h1 className="font-semibold text-2xl tracking-tight">Notas</h1>
        <p className="text-muted-foreground text-sm">
          Selecciona una sección para registrar notas.
        </p>
      </div>
      <SectionSelectionTable
        q={q}
        pageSize={pageSize}
        onQueryChange={onQueryChange}
        onPageSizeChange={onPageSizeChange}
        onSelectSection={handleSelectSection}
      />
    </div>
  );
}
