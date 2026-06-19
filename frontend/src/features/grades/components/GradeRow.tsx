import { RefreshCw, Save } from "lucide-react";
import { useEffect, useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { TableCell, TableRow } from "@/components/ui/table";
import { cn } from "@/core/utils/cn";
import type { Evaluation } from "@/gen/grades/v1/grades_pb";
import { gradeValueSchema } from "../gradeValue";
import type { CellVM } from "../hooks/useSectionGrid";
import { mapGradeWriteError } from "./errorMapping";

/** Per-cell save status within a row. */
type CellSaveStatus = "idle" | "saved" | "failed" | "conflict";

interface GradeRowProps {
  sectionEnrollmentId: string;
  studentId: string;
  displayName: string;
  /** Enrollment status: "in_progress", "withdrawn", "passed", "failed". */
  status: string;
  /** Ordered evaluations (columns). */
  evaluations: Evaluation[];
  /** Current cell values for this row. */
  cells: Map<string, CellVM>;
  /**
   * Issues a write call for a single cell.
   * Dispatches RecordGrade or OverrideGrade depending on caller's permission.
   * Throws on error.
   */
  onSaveCell: (params: {
    evaluationId: string;
    sectionEnrollmentId: string;
    value: string;
    expectedVersion?: number;
  }) => Promise<{ id: string; version: number; value: string }>;
  /**
   * Called after a conflict (CodeAborted) on this row.
   * Returns fresh cell data for ALL cells in the row so the caller can merge.
   */
  onConflictRefetch: (
    sectionEnrollmentId: string,
  ) => Promise<Map<string, CellVM>>;
  /**
   * Called when cells in this row are successfully saved or conflict-merged,
   * so the parent VM can be updated.
   */
  onCellsUpdated: (
    sectionEnrollmentId: string,
    updatedCells: Map<string, CellVM>,
  ) => void;
}

/**
 * Renders one student row in the grade recording grid.
 *
 * Behaviour:
 * - Withdrawn rows: grade inputs are disabled, Guardar is hidden.
 * - Per-row save: "Guardar" fans out one write per edited cell via Promise.allSettled.
 * - Partial failure: failed/conflict cells stay editable; succeeded cells commit new version.
 * - Conflict (CodeAborted): triggers row-scoped refetch + merge + inline message.
 * - "Reintentar" retries only non-succeeded cells.
 * - Validation (Zod): on blur, rejects out-of-range or multi-decimal values.
 */
export function GradeRow({
  sectionEnrollmentId,
  displayName,
  status,
  evaluations,
  cells,
  onSaveCell,
  onConflictRefetch,
  onCellsUpdated,
}: GradeRowProps) {
  const isWithdrawn = status === "withdrawn";

  // Local draft values: evaluationId → string input value
  const [drafts, setDrafts] = useState<Map<string, string>>(() => {
    const m = new Map<string, string>();
    for (const ev of evaluations) {
      m.set(ev.id, cells.get(ev.id)?.value ?? "");
    }
    return m;
  });

  // Per-cell validation errors
  const [validationErrors, setValidationErrors] = useState<Map<string, string>>(
    new Map(),
  );

  // Per-cell save status
  const [cellStatuses, setCellStatuses] = useState<Map<string, CellSaveStatus>>(
    () => {
      const m = new Map<string, CellSaveStatus>();
      for (const ev of evaluations) {
        m.set(ev.id, "idle");
      }
      return m;
    },
  );

  // Row-level conflict message (shown after CodeAborted)
  const [conflictMessage, setConflictMessage] = useState<string | null>(null);

  // True while the row save (or retry) is in flight
  const [isSaving, setIsSaving] = useState(false);

  // Track which cells have succeeded (so retry skips them)
  const [succeededCells, setSucceededCells] = useState<Set<string>>(new Set());

  // When the parent updates `cells` after a conflict refetch, re-sync drafts for any cell
  // whose status is "conflict" to the fresh server value. Cells that are idle, saved, or
  // in-progress (being actively edited) are left untouched so in-progress edits are not
  // clobbered. This effect runs ONLY when `cells` identity changes (parent merge completed).
  useEffect(() => {
    setCellStatuses((prevStatuses) => {
      const hasConflict = Array.from(prevStatuses.values()).some(
        (s) => s === "conflict",
      );
      if (!hasConflict) return prevStatuses;
      setDrafts((prevDrafts) => {
        const nextDrafts = new Map(prevDrafts);
        for (const [evId, status] of prevStatuses) {
          if (status === "conflict") {
            const freshValue = cells.get(evId)?.value ?? "";
            nextDrafts.set(evId, freshValue);
          }
        }
        return nextDrafts;
      });
      return prevStatuses;
    });
  }, [cells]);

  const hasValidationErrors =
    validationErrors.size > 0 &&
    Array.from(validationErrors.values()).some((e) => e !== "");

  const handleBlur = (evaluationId: string, value: string) => {
    if (value === "") {
      // Empty is allowed (no grade yet)
      setValidationErrors((prev) => {
        const next = new Map(prev);
        next.delete(evaluationId);
        return next;
      });
      return;
    }

    const result = gradeValueSchema.safeParse(value);
    if (!result.success) {
      const message = result.error.issues[0]?.message ?? "Valor inválido";
      setValidationErrors((prev) => new Map(prev).set(evaluationId, message));
    } else {
      setValidationErrors((prev) => {
        const next = new Map(prev);
        next.delete(evaluationId);
        return next;
      });
    }
  };

  const handleChange = (evaluationId: string, value: string) => {
    setDrafts((prev) => new Map(prev).set(evaluationId, value));
    // Clear the cell's status and error when the user edits it
    setCellStatuses((prev) => new Map(prev).set(evaluationId, "idle"));
    setValidationErrors((prev) => {
      const next = new Map(prev);
      next.delete(evaluationId);
      return next;
    });
  };

  /**
   * Determines which cells need to be saved in a given pass.
   * In the initial save, all cells with non-empty drafts that differ from the current
   * server value and that haven't already succeeded.
   * In retry, only non-succeeded cells that have a non-empty draft.
   */
  const getPendingCells = (
    skipSucceeded: Set<string>,
  ): Array<{
    evaluationId: string;
    value: string;
    expectedVersion?: number;
  }> => {
    const pending: Array<{
      evaluationId: string;
      value: string;
      expectedVersion?: number;
    }> = [];

    for (const ev of evaluations) {
      if (skipSucceeded.has(ev.id)) continue;

      const draft = drafts.get(ev.id) ?? "";
      if (draft === "") continue;

      // Validate before including
      const parseResult = gradeValueSchema.safeParse(draft);
      if (!parseResult.success) continue;

      const cell = cells.get(ev.id);
      const currentVersion = cell?.version ?? 0;
      const expectedVersion = currentVersion > 0 ? currentVersion : undefined;

      pending.push({
        evaluationId: ev.id,
        value: draft,
        expectedVersion,
      });
    }
    return pending;
  };

  const executeSave = async (skipSucceeded: Set<string>) => {
    const pending = getPendingCells(skipSucceeded);
    if (pending.length === 0) return;

    setIsSaving(true);
    setConflictMessage(null);

    const results = await Promise.allSettled(
      pending.map((cell) =>
        onSaveCell({
          evaluationId: cell.evaluationId,
          sectionEnrollmentId,
          value: cell.value,
          expectedVersion: cell.expectedVersion,
        }),
      ),
    );

    const newSucceeded = new Set(skipSucceeded);
    const newCellStatuses = new Map(cellStatuses);
    const updatedCells = new Map(cells);
    let hadConflict = false;

    for (let i = 0; i < results.length; i++) {
      const result = results[i];
      const cell = pending[i];

      if (result.status === "fulfilled") {
        newSucceeded.add(cell.evaluationId);
        newCellStatuses.set(cell.evaluationId, "saved");
        // Commit the new version into the cells map
        updatedCells.set(cell.evaluationId, {
          evaluationId: cell.evaluationId,
          value: result.value.value,
          version: result.value.version,
          gradeId: result.value.id,
        });
      } else {
        const kind = mapGradeWriteError(result.reason);
        if (kind === "conflict") {
          hadConflict = true;
          newCellStatuses.set(cell.evaluationId, "conflict");
        } else {
          newCellStatuses.set(cell.evaluationId, "failed");
        }
      }
    }

    setSucceededCells(newSucceeded);
    setCellStatuses(newCellStatuses);

    if (hadConflict) {
      setConflictMessage(
        "Otro usuario modificó esta nota. Recarga para ver el valor actualizado.",
      );
      // Row-scoped refetch: merge fresh versions for ALL cells in this row
      try {
        const freshCells = await onConflictRefetch(sectionEnrollmentId);
        // Merge: keep succeeded cell data (already committed), update others
        for (const [evId, freshCell] of freshCells) {
          if (!newSucceeded.has(evId)) {
            updatedCells.set(evId, freshCell);
          }
        }
      } catch {
        // Refetch failed; the conflict message is already showing
      }
    }

    onCellsUpdated(sectionEnrollmentId, updatedCells);
    setIsSaving(false);
  };

  const handleSave = () => {
    if (hasValidationErrors || isSaving) return;
    void executeSave(new Set());
  };

  const handleRetry = () => {
    if (isSaving) return;
    void executeSave(succeededCells);
  };

  const hasFailedCells = Array.from(cellStatuses.values()).some(
    (s) => s === "failed" || s === "conflict",
  );

  const hasPendingCells = evaluations.some((ev) => {
    const draft = drafts.get(ev.id) ?? "";
    return draft !== "" && !succeededCells.has(ev.id);
  });

  return (
    <TableRow>
      <TableCell className="font-medium">
        <div className="flex items-center gap-2">
          <span>{displayName}</span>
          {isWithdrawn && (
            <Badge variant="secondary" className="text-xs">
              Retirado
            </Badge>
          )}
        </div>
      </TableCell>

      {evaluations.map((ev) => {
        const draft = drafts.get(ev.id) ?? "";
        const validationError = validationErrors.get(ev.id);
        const cellStatus = cellStatuses.get(ev.id) ?? "idle";

        return (
          <TableCell key={ev.id} className="min-w-[100px]">
            <div className="flex flex-col gap-1">
              <Input
                type="text"
                inputMode="decimal"
                value={draft}
                disabled={isWithdrawn || isSaving}
                onChange={(e) => handleChange(ev.id, e.target.value)}
                onBlur={(e) => handleBlur(ev.id, e.target.value)}
                className={cn(
                  "h-8 w-20 text-sm",
                  validationError && "border-destructive",
                  cellStatus === "saved" && "border-green-500",
                  (cellStatus === "failed" || cellStatus === "conflict") &&
                    "border-destructive",
                )}
                aria-label={`Nota para evaluación ${ev.position}`}
                aria-invalid={!!validationError}
              />
              {/* Reserve layout space for error/status messages */}
              <div className="min-h-[1rem] text-xs">
                {validationError && (
                  <span className="text-destructive">{validationError}</span>
                )}
                {!validationError && cellStatus === "saved" && (
                  <span className="text-green-600">Guardado</span>
                )}
                {!validationError &&
                  (cellStatus === "failed" || cellStatus === "conflict") && (
                    <span className="text-destructive">Error</span>
                  )}
              </div>
            </div>
          </TableCell>
        );
      })}

      {/* Row actions column */}
      <TableCell>
        <div className="flex flex-col gap-1">
          {!isWithdrawn && (
            <div className="flex items-center gap-2">
              {hasPendingCells && (
                <Button
                  size="sm"
                  onClick={handleSave}
                  disabled={hasValidationErrors || isSaving}
                  className="gap-1"
                >
                  <Save
                    className="size-3"
                    data-icon="inline-start"
                    aria-hidden
                  />
                  Guardar
                </Button>
              )}
              {hasFailedCells && (
                <Button
                  size="sm"
                  variant="outline"
                  onClick={handleRetry}
                  disabled={isSaving}
                  className="gap-1"
                >
                  <RefreshCw
                    className="size-3"
                    data-icon="inline-start"
                    aria-hidden
                  />
                  Reintentar
                </Button>
              )}
            </div>
          )}
          {conflictMessage && (
            <p className="text-destructive text-xs max-w-[200px]">
              {conflictMessage}
            </p>
          )}
        </div>
      </TableCell>
    </TableRow>
  );
}
