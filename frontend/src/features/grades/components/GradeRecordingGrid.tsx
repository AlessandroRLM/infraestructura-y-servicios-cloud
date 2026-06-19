import { RefreshCw } from "lucide-react";
import { useCallback, useState } from "react";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { hasPermission, useSession } from "@/features/auth";
import type { TeachingSection } from "@/gen/catalog/v1/catalog_pb";
import { useOverrideGrade } from "../hooks/useOverrideGrade";
import { useRecordGrade } from "../hooks/useRecordGrade";
import type { CellVM } from "../hooks/useSectionGrid";
import { useSectionGrid } from "../hooks/useSectionGrid";
import { AdminSchemeButton } from "./AdminSchemeButton";
import { GradeRow } from "./GradeRow";

interface GradeRecordingGridProps {
  /** The selected section to record grades for. */
  section: TeachingSection;
  /** Called when the user wants to go back to the section selection table. */
  onBack: () => void;
}

/**
 * Orchestrator component for the grade recording grid.
 *
 * Manages:
 * - Composing useSectionGrid (roster × evaluations × grades × display names)
 * - Local mutable copy of cells (so row updates don't require a full refetch)
 * - Write dispatch: grades.override → OverrideGrade, else → RecordGrade
 * - Empty states: no scheme, no enrolled students
 * - Admin "Administrar Notas" button in the top-right (grades.override only)
 */
export function GradeRecordingGrid({
  section,
  onBack,
}: GradeRecordingGridProps) {
  const session = useSession();
  const isAdmin = hasPermission(session, "grades.override");

  const { evaluations, rows, isLoading, isError, mergeRowGrades } =
    useSectionGrid(section.id, section.courseId);

  const { record } = useRecordGrade();
  const { override } = useOverrideGrade();

  // Local mutable cells state (seeded from the VM, updated on save without full refetch)
  const [localCells, setLocalCells] = useState<
    Map<string, Map<string, CellVM>>
  >(new Map());

  // Initialize localCells once data loads
  const [initialized, setInitialized] = useState(false);
  if (!isLoading && !isError && rows.length > 0 && !initialized) {
    const m = new Map<string, Map<string, CellVM>>();
    for (const row of rows) {
      m.set(row.sectionEnrollmentId, new Map(row.cells));
    }
    setLocalCells(m);
    setInitialized(true);
  }

  const handleSaveCell = useCallback(
    async (params: {
      evaluationId: string;
      sectionEnrollmentId: string;
      value: string;
      expectedVersion?: number;
    }): Promise<{ id: string; version: number; value: string }> => {
      const grade = isAdmin
        ? await override({
            evaluationId: params.evaluationId,
            sectionEnrollmentId: params.sectionEnrollmentId,
            value: params.value,
            expectedVersion: params.expectedVersion,
          })
        : await record({
            evaluationId: params.evaluationId,
            sectionEnrollmentId: params.sectionEnrollmentId,
            value: params.value,
            expectedVersion: params.expectedVersion,
          });
      return { id: grade.id, version: grade.version, value: grade.value };
    },
    [isAdmin, override, record],
  );

  const handleCellsUpdated = useCallback(
    (sectionEnrollmentId: string, updatedCells: Map<string, CellVM>) => {
      setLocalCells((prev) => {
        const next = new Map(prev);
        next.set(sectionEnrollmentId, updatedCells);
        return next;
      });
    },
    [],
  );

  const handleConflictRefetch = useCallback(
    (sectionEnrollmentId: string) => mergeRowGrades(sectionEnrollmentId),
    [mergeRowGrades],
  );

  return (
    <div className="flex flex-col gap-4">
      {/* Header: back button + section info + admin button */}
      <div className="flex items-center justify-between">
        <div className="flex flex-col gap-1">
          <Button variant="ghost" size="sm" onClick={onBack} className="w-fit">
            ← Volver a secciones
          </Button>
          <h2 className="font-semibold text-lg">
            {section.courseName}{" "}
            <span className="text-muted-foreground font-normal text-sm">
              {section.courseCode} · {section.periodYear} Sem.{" "}
              {section.periodTerm}
            </span>
          </h2>
        </div>
        {isAdmin && <AdminSchemeButton courseId={section.courseId} />}
      </div>

      {/* Loading state */}
      {isLoading && (
        <div
          role="status"
          className="flex flex-col gap-2"
          aria-busy="true"
          aria-label="Cargando notas"
        >
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-10 w-full" />
          ))}
        </div>
      )}

      {/* Error state */}
      {!isLoading && isError && (
        <div
          className="rounded-md border border-destructive/50 p-4"
          role="alert"
        >
          <p className="text-destructive text-sm font-medium">
            No se pudieron cargar las notas de esta sección.
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-3 gap-2"
            onClick={() => window.location.reload()}
          >
            <RefreshCw className="size-4" aria-hidden />
            Reintentar
          </Button>
        </div>
      )}

      {/* Empty state: no evaluation scheme */}
      {!isLoading && !isError && evaluations.length === 0 && (
        <div className="flex flex-col items-center justify-center gap-4 rounded-md border border-dashed p-12 text-center">
          <p className="text-muted-foreground text-sm">
            Esta sección no tiene esquema de evaluación.
          </p>
          {isAdmin && (
            <AdminSchemeButton courseId={section.courseId} variant="outline" />
          )}
        </div>
      )}

      {/* Empty state: no enrolled students */}
      {!isLoading &&
        !isError &&
        evaluations.length > 0 &&
        rows.length === 0 && (
          <div className="flex flex-col items-center justify-center gap-4 rounded-md border border-dashed p-12 text-center">
            <p className="text-muted-foreground text-sm">
              No hay estudiantes inscritos en esta sección.
            </p>
          </div>
        )}

      {/* Grade grid */}
      {!isLoading && !isError && evaluations.length > 0 && rows.length > 0 && (
        <div className="rounded-md border overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="min-w-[180px]">Estudiante</TableHead>
                {evaluations.map((ev) => (
                  <TableHead key={ev.id} className="min-w-[100px]">
                    Eval. {ev.position}
                    <span className="block text-xs text-muted-foreground font-normal">
                      {(parseFloat(ev.weight) * 100).toFixed(0)}%
                    </span>
                  </TableHead>
                ))}
                <TableHead className="min-w-[140px]">Acciones</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row) => {
                const rowCells =
                  localCells.get(row.sectionEnrollmentId) ?? row.cells;
                return (
                  <GradeRow
                    key={row.sectionEnrollmentId}
                    sectionEnrollmentId={row.sectionEnrollmentId}
                    studentId={row.studentId}
                    displayName={row.displayName}
                    status={row.status}
                    evaluations={evaluations}
                    cells={rowCells}
                    onSaveCell={handleSaveCell}
                    onConflictRefetch={handleConflictRefetch}
                    onCellsUpdated={handleCellsUpdated}
                  />
                );
              })}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
