import { ArrowLeft } from "lucide-react";
import { useState } from "react";
import { SectionSelectionTable } from "@/features/grades";
import type { TeachingSection } from "@/gen/catalog/v1/catalog_pb";
import { SectionEnrollmentsTable } from "./SectionEnrollmentsTable";

interface PageState {
  view: "select" | "roster";
  section: TeachingSection | null;
}

/**
 * Admin section-enrollments page.
 *
 * Flow:
 *  1. Section selection view: shows SectionSelectionTable.
 *     Clicking a row transitions to the roster view.
 *  2. Roster view: shows SectionEnrollmentsTable for the selected section
 *     with "Volver a secciones" to go back.
 *
 * Search state (q, pageSize) is local to this component — the route already
 * owns the URL params via validateSearch, and this component reads them from
 * props. No sub-route navigation needed: the selected section stays in local
 * state (refresh-proof via URL params is not required for this admin flow).
 */
interface SectionEnrollmentsPageProps {
  q: string;
  pageSize: number;
  onQueryChange: (v: string) => void;
  onPageSizeChange: (n: number) => void;
}

export function SectionEnrollmentsPage({
  q,
  pageSize,
  onQueryChange,
  onPageSizeChange,
}: SectionEnrollmentsPageProps) {
  const [pageState, setPageState] = useState<PageState>({
    view: "select",
    section: null,
  });

  if (pageState.view === "roster" && pageState.section) {
    const section = pageState.section;
    return (
      <div className="flex flex-col gap-6">
        <button
          type="button"
          className="inline-flex items-center gap-1.5 text-muted-foreground text-sm transition-colors hover:text-foreground w-fit"
          onClick={() => setPageState({ view: "select", section: null })}
        >
          <ArrowLeft className="size-4" aria-hidden />
          Volver a secciones
        </button>

        <div className="flex flex-col gap-1">
          <h1 className="font-semibold text-2xl tracking-tight">
            Inscripciones a secciones
          </h1>
          <p className="text-muted-foreground text-sm">
            {section.courseName} — {section.courseCode} · {section.periodYear}{" "}
            Semestre {section.periodTerm}
          </p>
        </div>

        <SectionEnrollmentsTable sectionId={section.id} />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-1">
        <h1 className="font-semibold text-2xl tracking-tight">
          Inscripciones a secciones
        </h1>
        <p className="text-muted-foreground text-sm">
          Selecciona una sección para gestionar sus inscripciones.
        </p>
      </div>

      <SectionSelectionTable
        q={q}
        pageSize={pageSize}
        onQueryChange={onQueryChange}
        onPageSizeChange={onPageSizeChange}
        onSelectSection={(section) => setPageState({ view: "roster", section })}
      />
    </div>
  );
}
