import { Loader2, RefreshCw } from "lucide-react";
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
import { useOwnSections } from "@/core/catalog";
import { PageSizeSelector, SearchInput } from "@/core/components";
import type { TeachingSection } from "@/gen/catalog/v1/catalog_pb";

const SECTIONS_SEARCH_DEBOUNCE_MS = 300;

interface SectionSelectionTableProps {
  /** Current search query; drives the SearchInput value and the RPC query param. */
  q: string;
  /** Current page size; drives the PageSizeSelector and the RPC pageSize param. */
  pageSize: number;
  /** Called with the debounced query value after the user types. */
  onQueryChange: (v: string) => void;
  /** Called when the user picks a different page size. */
  onPageSizeChange: (n: number) => void;
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
 * q and pageSize are owned by the caller (URL-synced via the route); this
 * component is purely presentational with respect to list-state.
 *
 * Uses the infinite-query pattern from ProgramsTable for "Cargar más".
 */
export function SectionSelectionTable({
  q,
  pageSize,
  onQueryChange,
  onPageSizeChange,
  onSelectSection,
}: SectionSelectionTableProps) {
  const {
    sections,
    isLoading,
    isError,
    refetch,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useOwnSections({ query: q, pageSize });

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-2">
        <SearchInput
          value={q}
          onChange={onQueryChange}
          debounceMs={SECTIONS_SEARCH_DEBOUNCE_MS}
          placeholder="Buscar asignatura…"
          className="max-w-sm"
        />
        <PageSizeSelector value={pageSize} onChange={onPageSizeChange} />
      </div>
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
            {q !== ""
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
