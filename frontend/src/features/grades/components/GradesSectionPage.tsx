import { Link } from "@tanstack/react-router";
import { ArrowLeft } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { useOwnSections } from "@/core/catalog";
import type { TeachingSection } from "@/gen/catalog/v1/catalog_pb";
import { GradeRecordingGrid } from "./GradeRecordingGrid";

interface GradesSectionPageProps {
  /** The section ID from the URL param $sectionId. */
  sectionId: string;
  /**
   * Router navigation state from the parent route's location.
   * If the user navigated by clicking a row, the full TeachingSection is
   * present here as `locationState.section` — no extra fetch needed.
   * On hard refresh or deep-link, locationState is the bare history key object
   * and section is absent; the page resolves via useOwnSections instead.
   */
  locationState: unknown;
  /** Called when the user clicks "Volver a secciones". */
  onBack: () => void;
}

/**
 * Resolves the TeachingSection for the given sectionId and renders
 * GradeRecordingGrid once the section is available.
 *
 * Resolution strategy (in order):
 * 1. Router navigation state — instant, no extra fetch (normal click-through).
 * 2. useOwnSections — resolves by id from the already-cached list (deep-link / refresh).
 * 3. Not found in loaded data → graceful "No se pudo cargar la sección" with back link.
 *    Does NOT loop-fetch: if the section is not in the user's list, it never will be.
 */
export function GradesSectionPage({
  sectionId,
  locationState,
  onBack,
}: GradesSectionPageProps) {
  // Attempt 1: section passed via router navigation state on click-through.
  const stateSection = extractSectionFromState(locationState, sectionId);

  // Attempt 2: deep-link / refresh — resolve from the section list.
  // Disabled when stateSection is available to avoid a redundant ListOwnSections RPC.
  const { sections, isLoading } = useOwnSections({ enabled: !stateSection });

  const resolvedSection =
    stateSection ?? sections.find((s) => s.id === sectionId) ?? null;

  if (stateSection) {
    return (
      <GradeRecordingGrid
        key={stateSection.id}
        section={stateSection}
        onBack={onBack}
      />
    );
  }

  if (isLoading) {
    return (
      <div
        role="status"
        className="flex flex-col gap-2"
        aria-busy="true"
        aria-label="Cargando sección"
      >
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-10 w-full" />
        ))}
      </div>
    );
  }

  if (!resolvedSection) {
    return (
      <div className="flex flex-col gap-4">
        <p className="text-muted-foreground text-sm">
          No se pudo cargar la sección.
        </p>
        <Link
          to="/admin/grades"
          className="inline-flex items-center gap-1.5 text-muted-foreground text-sm transition-colors hover:text-foreground w-fit"
        >
          <ArrowLeft className="size-4" aria-hidden />
          Volver a secciones
        </Link>
      </div>
    );
  }

  return (
    <GradeRecordingGrid
      key={resolvedSection.id}
      section={resolvedSection}
      onBack={onBack}
    />
  );
}

/**
 * Narrows the raw router location state to a TeachingSection for the given id.
 * Returns null when the state is absent, malformed, or belongs to a different section.
 */
function extractSectionFromState(
  state: unknown,
  sectionId: string,
): TeachingSection | null {
  if (
    state !== null &&
    typeof state === "object" &&
    "section" in state &&
    state.section !== null &&
    typeof state.section === "object" &&
    "id" in state.section &&
    (state.section as { id: unknown }).id === sectionId
  ) {
    return state.section as TeachingSection;
  }
  return null;
}
