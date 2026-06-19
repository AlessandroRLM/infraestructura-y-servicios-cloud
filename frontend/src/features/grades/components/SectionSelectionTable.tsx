import { Loader2, RefreshCw } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { SearchInput } from "@/core/components";
import type { TeachingSection } from "@/gen/catalog/v1/catalog_pb";
import { useOwnSections } from "../hooks/useOwnSections";

const SECTIONS_SEARCH_DEBOUNCE_MS = 300;

interface SectionSelectionTableProps {
  /** Called when the user clicks a section row. */
  onSelectSection: (section: TeachingSection) => void;
}

/**
 * Presentational table that lists teaching sections with three columns:
 * Asignatura (course name), Sección (section id), Período (year·term).
 * Clicking a row calls onSelectSection.
 *
 * Columns are intentionally limited to the spec:
 * - Asignatura, Sección, Período — no cupo/enrolled column.
 *
 * Uses the infinite-query pattern from ProgramsTable for "Cargar más".
 */
export function SectionSelectionTable({
  onSelectSection,
}: SectionSelectionTableProps) {
  /**
   * Committed (debounced) query value.
   * SearchInput owns the raw keystroke state internally; this state holds the
   * last value emitted by SearchInput.onChange (after the debounce window).
   * Passing it back as `value` keeps SearchInput in sync after external resets.
   */
  const [query, setQuery] = useState("");

  const {
    sections,
    isLoading,
    isError,
    refetch,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useOwnSections({ query });

  return (
    <div className="flex flex-col gap-4">
      <SearchInput
        value={query}
        onChange={setQuery}
        debounceMs={SECTIONS_SEARCH_DEBOUNCE_MS}
        placeholder="Buscar asignatura…"
        className="max-w-sm"
      />
      {isLoading && (
        <div
          role="status"
          className="flex flex-col gap-2"
          aria-busy="true"
          aria-label="Cargando secciones"
        >
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-10 w-full" />
          ))}
        </div>
      )}

      {!isLoading && isError && (
        <div
          className="rounded-md border border-destructive/50 p-4"
          role="alert"
        >
          <p className="text-destructive text-sm font-medium">
            No se pudo cargar la lista de secciones.
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-3 gap-2"
            onClick={() => refetch()}
          >
            <RefreshCw className="size-4" aria-hidden />
            Reintentar
          </Button>
        </div>
      )}

      {!isLoading && !isError && sections.length === 0 && (
        <div className="flex flex-col items-center justify-center gap-4 rounded-md border border-dashed p-12 text-center">
          <p className="text-muted-foreground text-sm">
            {query !== ""
              ? "No se encontraron secciones para la búsqueda."
              : "No hay secciones asignadas."}
          </p>
        </div>
      )}

      {!isLoading && !isError && sections.length > 0 && (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Asignatura</TableHead>
                <TableHead>Sección</TableHead>
                <TableHead>Período</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {sections.map((section) => (
                <TableRow
                  key={section.id}
                  className="cursor-pointer"
                  onClick={() => onSelectSection(section)}
                  role="button"
                  tabIndex={0}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" || e.key === " ") {
                      e.preventDefault();
                      onSelectSection(section);
                    }
                  }}
                  aria-label={`Seleccionar sección ${section.courseCode} ${section.id}`}
                >
                  <TableCell className="font-medium">
                    {section.courseName}
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {section.courseCode}
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {section.periodYear} · Semestre {section.periodTerm}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {hasNextPage && (
        <div className="flex justify-center">
          <Button
            variant="outline"
            onClick={async () => {
              try {
                await fetchNextPage();
              } catch {
                toast.error("No se pudieron cargar más secciones.");
              }
            }}
            disabled={isFetchingNextPage}
            className="gap-2"
          >
            {isFetchingNextPage && (
              <Loader2
                data-icon="inline-start"
                className="animate-spin"
                aria-hidden
              />
            )}
            Cargar más
          </Button>
        </div>
      )}
    </div>
  );
}
