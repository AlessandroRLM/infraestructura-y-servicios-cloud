import { Loader2, RefreshCw } from "lucide-react";
import { toast } from "sonner";
import { Accordion } from "@/components/ui/accordion";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Route } from "@/routes/_authenticated/grades";
import {
  buildProgramOptions,
  useOwnEnrollmentsForFilter,
} from "../hooks/useOwnEnrollmentsForFilter";
import { useOwnGradePeriods } from "../hooks/useOwnGradePeriods";
import { useOwnGrades } from "../hooks/useOwnGrades";
import { GradeSectionGroup } from "./GradeSectionGroup";
import { GradesFilterBar } from "./GradesFilterBar";

/**
 * Main view for the student "Mis notas" page.
 * Renders the período/carrera filter bar, the accordion of section groups,
 * and loading/empty/error states mirroring the academics pattern.
 */
export function OwnGradesView() {
  const { period, program, pageSize } = Route.useSearch();

  const {
    groups,
    rawGrades,
    isLoading,
    isError,
    refetch,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useOwnGrades(period, program, pageSize);

  const { periods, isLoading: isLoadingPeriods } = useOwnGradePeriods();

  const { programIds, isLoading: isLoadingEnrollments } =
    useOwnEnrollmentsForFilter();

  // Build a programId → programName map from the loaded OwnGrade rows.
  // Each grade carries both programId and programName (always present per backend spec).
  const programNameMap = new Map<string, string>();
  for (const grade of rawGrades) {
    if (grade.programId && grade.programName) {
      programNameMap.set(grade.programId, grade.programName);
    }
  }

  const programOptions = buildProgramOptions(programIds, programNameMap);

  const hasActiveFilter = Boolean(period || program);

  return (
    <div className="flex flex-col gap-4">
      <GradesFilterBar
        periods={periods}
        programs={programOptions}
        isLoadingPeriods={isLoadingPeriods || isLoadingEnrollments}
      />

      {isLoading && (
        <div
          role="status"
          className="space-y-2"
          aria-busy="true"
          aria-label="Cargando notas"
        >
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-14 w-full" />
          ))}
        </div>
      )}

      {!isLoading && isError && (
        <div
          className="rounded-md border border-destructive/50 p-4"
          role="alert"
        >
          <p className="text-destructive text-sm font-medium">
            No se pudieron cargar las notas.
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

      {!isLoading && !isError && groups.length === 0 && (
        <div className="flex flex-col items-center justify-center gap-4 rounded-md border border-dashed p-12 text-center">
          <p className="text-muted-foreground text-sm">
            {hasActiveFilter
              ? "No hay notas que coincidan con los filtros activos."
              : "Todavía no tienes notas registradas."}
          </p>
        </div>
      )}

      {!isLoading && !isError && groups.length > 0 && (
        <div className="rounded-md border">
          <Accordion type="multiple">
            {groups.map((group) => (
              <GradeSectionGroup
                key={group.sectionEnrollmentId}
                group={group}
              />
            ))}
          </Accordion>
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
                toast.error("No se pudieron cargar más notas.");
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
