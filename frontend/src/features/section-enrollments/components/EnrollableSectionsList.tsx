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
import { academicPeriodLabel } from "@/core/catalog";
import { mapEnrollOwnSectionError } from "../hooks/errorMapping";
import { useEnrollableSections } from "../hooks/useEnrollableSections";
import { useEnrollOwnSection } from "../hooks/useEnrollOwnSection";

/**
 * Presentational list of sections the authenticated student can self-enroll into.
 * Each row exposes an "Inscribirme" action that calls enrollOwnSection.
 * On success shows a sonner toast and lets cache invalidation refresh both lists.
 */
export function EnrollableSectionsList() {
  const {
    sections,
    isLoading,
    isError,
    refetch,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useEnrollableSections();

  const mutation = useEnrollOwnSection();

  async function handleEnroll(sectionId: string, programId: string) {
    try {
      await mutation.mutateAsync({ sectionId, programId });
      toast.success("Inscripción realizada con éxito.");
    } catch (err) {
      toast.error(mapEnrollOwnSectionError(err));
    }
  }

  return (
    <div className="flex flex-col gap-4">
      {isLoading && (
        <div
          role="status"
          className="space-y-2"
          aria-busy="true"
          aria-label="Cargando secciones disponibles"
        >
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      )}

      {!isLoading && isError && (
        <div
          className="rounded-md border border-destructive/50 p-4"
          role="alert"
        >
          <p className="text-destructive text-sm font-medium">
            No se pudieron cargar las secciones disponibles.
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
            No hay secciones disponibles para inscribirte.
          </p>
        </div>
      )}

      {!isLoading && !isError && sections.length > 0 && (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Asignatura</TableHead>
                <TableHead>Período</TableHead>
                <TableHead>Cupos</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {sections.map((section) => (
                <TableRow key={`${section.sectionId}-${section.programId}`}>
                  <TableCell className="font-medium">
                    {section.courseName} ({section.courseCode})
                  </TableCell>
                  <TableCell>
                    {academicPeriodLabel(
                      section.periodYear,
                      section.periodTerm,
                    )}
                  </TableCell>
                  <TableCell>{section.seatsAvailable}</TableCell>
                  <TableCell className="text-right">
                    <Button
                      size="sm"
                      disabled={mutation.isPending}
                      onClick={() =>
                        handleEnroll(section.sectionId, section.programId)
                      }
                    >
                      {mutation.isPending && (
                        <Loader2
                          data-icon="inline-start"
                          className="animate-spin"
                          aria-hidden
                        />
                      )}
                      Inscribirme
                    </Button>
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
                await fetchNextPage({ throwOnError: true });
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
