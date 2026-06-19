import { RefreshCw } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
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
import type { CellVM, RowVM } from "../hooks/useSectionGrid";
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
 * Merges session-level saved overrides into the server rows.
 * Override cells (from successful saves in this session) take precedence over
 * the server value so background refetches do not clobber saved grades.
 */
function mergeRowsWithOverrides(
  rows: RowVM[],
  overrides: Map<string, Map<string, CellVM>>,
): RowVM[] {
  if (overrides.size === 0) return rows;
  return rows.map((row) => {
    const rowOverrides = overrides.get(row.sectionEnrollmentId);
    if (!rowOverrides || rowOverrides.size === 0) return row;
    const mergedCells = new Map(row.cells);
    for (const [evalId, cell] of rowOverrides) {
      mergedCells.set(evalId, cell);
    }
    return { ...row, cells: mergedCells };
  });
}

/**
 * Orchestrator component for the grade recording grid.
 *
 * Manages:
 * - Composing useSectionGrid (roster × evaluations × grades × display names)
 * - Session-level overrides: cells saved this session are stored in `overrides`
 *   and merged over server rows via useMemo — background refetches never clobber edits.
 * - Conflict path: clears the stale row override before invalidating the grades cache
 *   so the fresh server value flows through without the stale override masking it.
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

  const { evaluations, rows, isLoading, isError, refetchGrades } =
    useSectionGrid(section.id, section.courseId);

  const { record } = useRecordGrade();
  const { override } = useOverrideGrade();

  // Session-level saved overrides: seId → (evalId → CellVM).
  // Saved cells are stored here so background refetches of `rows` do not
  // clobber values already committed this session.
  const [overrides, setOverrides] = useState<Map<string, Map<string, CellVM>>>(
    new Map(),
  );

  // Merge overrides into server rows — override wins on value + version.
  const displayRows = useMemo(
    () => mergeRowsWithOverrides(rows, overrides),
    [rows, overrides],
  );

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

      const saved = {
        id: grade.id,
        version: grade.version,
        value: grade.value,
      };

      // Record the saved cell in overrides so it survives background refetches.
      setOverrides((prev) => {
        const next = new Map(prev);
        const rowOverrides = new Map(
          next.get(params.sectionEnrollmentId) ?? [],
        );
        rowOverrides.set(params.evaluationId, {
          evaluationId: params.evaluationId,
          value: saved.value,
          version: saved.version,
          gradeId: saved.id,
        });
        next.set(params.sectionEnrollmentId, rowOverrides);
        return next;
      });

      return saved;
    },
    [isAdmin, override, record],
  );

  // Builds the per-row conflict handler for a given sectionEnrollmentId.
  // On conflict: clears the stale row override so the fresh cache value wins,
  // then invalidates the grades query so TanStack Query re-fetches.
  const makeConflictRefetch = useCallback(
    (sectionEnrollmentId: string) => (): Promise<void> => {
      // Remove the stale override for this row before invalidating — without
      // this the override map would mask the fresh server value after re-fetch.
      setOverrides((prev) => {
        const next = new Map(prev);
        next.delete(sectionEnrollmentId);
        return next;
      });
      return refetchGrades();
    },
    [refetchGrades],
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
        {isAdmin && (
          <AdminSchemeButton
            courseId={section.courseId}
            courseLabel={`${section.courseCode} — ${section.courseName}`}
          />
        )}
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
            <AdminSchemeButton
              courseId={section.courseId}
              courseLabel={`${section.courseCode} — ${section.courseName}`}
              variant="outline"
            />
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
              {displayRows.map((row) => (
                <GradeRow
                  key={row.sectionEnrollmentId}
                  sectionEnrollmentId={row.sectionEnrollmentId}
                  displayName={row.displayName}
                  status={row.status}
                  evaluations={evaluations}
                  cells={row.cells}
                  onSaveCell={handleSaveCell}
                  onConflictRefetch={makeConflictRefetch(
                    row.sectionEnrollmentId,
                  )}
                />
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
