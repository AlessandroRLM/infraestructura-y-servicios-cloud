import { RefreshCw, Save } from "lucide-react";
import { useState } from "react";
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
type CellSaveStatus = "idle" | "saved" | "failed";

interface GradeRowProps {
  sectionEnrollmentId: string;
  displayName: string;
  /** Enrollment status: "in_progress", "withdrawn", "passed", "failed". */
  status: string;
  /** Ordered evaluations (columns). */
  evaluations: Evaluation[];
  /** Current cell values for this row (from displayRows — includes session overrides). */
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
   * Invalidates the grades query cache so TanStack Query re-fetches and rows
   * update automatically. The caller is responsible for clearing stale overrides.
   */
  onConflictRefetch: () => Promise<void>;
}

/**
 * Renders one student row in the grade recording grid.
 *
 * Behaviour:
 * - Withdrawn rows: grade inputs are disabled, Guardar is hidden.
 * - Per-row save: "Guardar" fans out one write per edited cell via Promise.allSettled.
 * - Partial failure: failed cells stay editable; succeeded cells commit new version.
 * - Conflict (CodeAborted): triggers cache invalidation via onConflictRefetch; cell resets
 *   to idle showing the fresh server value; conflict message shown transiently.
 * - "Reintentar" retries only non-succeeded cells.
 * - Validation (Zod): on blur, rejects out-of-range or multi-decimal values.
 * - drafts is a SPARSE overlay: only cells the user has actively edited are present.
 *   The displayed value derives as: draft (if present) else server value from cells prop.
 */
export function GradeRow({
  sectionEnrollmentId,
  displayName,
  status,
  evaluations,
  cells,
  onSaveCell,
  onConflictRefetch,
}: GradeRowProps) {
  const isWithdrawn = status === "withdrawn";

  // Sparse draft overlay: only contains evaluationIds the user has actively typed into.
  const [drafts, setDrafts] = useState<Map<string, string>>(new Map());

  // Per-cell validation errors
  const [validationErrors, setValidationErrors] = useState<Map<string, string>>(
    new Map(),
  );

  // Per-cell save status — "conflict" removed: conflicts reset cells to idle immediately
  const [cellStatuses, setCellStatuses] = useState<Map<string, CellSaveStatus>>(
    new Map(),
  );

  // Row-level conflict message (shown after CodeAborted, cleared on next save)
  const [conflictMessage, setConflictMessage] = useState<string | null>(null);

  // True while the row save (or retry) is in flight
  const [isSaving, setIsSaving] = useState(false);

  // Track which cells have succeeded (so retry skips them)
  const [succeededCells, setSucceededCells] = useState<Set<string>>(new Set());

  const hasValidationErrors =
    validationErrors.size > 0 &&
    Array.from(validationErrors.values()).some((e) => e !== "");

  const handleBlur = (evaluationId: string, value: string) => {
    if (value === "") {
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
    setCellStatuses((prev) => new Map(prev).set(evaluationId, "idle"));
    setValidationErrors((prev) => {
      const next = new Map(prev);
      next.delete(evaluationId);
      return next;
    });
  };

  /**
   * Determines which cells need to be saved in a given pass.
   * Skips: already succeeded, empty draft, invalid draft.
   * Uses the sparse draft overlay — cells without a draft entry are not pending.
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

      const draft = drafts.get(ev.id);
      // No draft entry means the user hasn't touched this cell — skip it
      if (draft === undefined || draft === "") continue;

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
    const newDrafts = new Map(drafts);
    let hadConflict = false;

    for (let i = 0; i < results.length; i++) {
      const result = results[i];
      const cell = pending[i];

      if (result.status === "fulfilled") {
        newSucceeded.add(cell.evaluationId);
        newCellStatuses.set(cell.evaluationId, "saved");
        // Remove draft: input falls back to the server value (provided by onSaveCell
        // updating the parent's overrides, which flows back via cells prop)
        newDrafts.delete(cell.evaluationId);
      } else {
        const kind = mapGradeWriteError(result.reason);
        if (kind === "conflict") {
          hadConflict = true;
          // Conflict: reset to idle immediately — do NOT leave cell stuck in red
          newCellStatuses.set(cell.evaluationId, "idle");
          newDrafts.delete(cell.evaluationId);
        } else {
          newCellStatuses.set(cell.evaluationId, "failed");
        }
      }
    }

    setSucceededCells(newSucceeded);
    setCellStatuses(newCellStatuses);
    setDrafts(newDrafts);

    if (hadConflict) {
      setConflictMessage(
        "Otro usuario modificó esta nota. Se actualizó al valor más reciente.",
      );
      // Invalidate the grades cache: TanStack Query re-fetches and rows update
      // automatically, showing the authoritative server value.
      try {
        await onConflictRefetch();
      } catch {
        // Cache invalidation failed; the conflict message is still shown
      }
    }

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
    (s) => s === "failed",
  );

  const hasPendingCells = evaluations.some((ev) => {
    const draft = drafts.get(ev.id);
    return draft !== undefined && draft !== "" && !succeededCells.has(ev.id);
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
        // Derive displayed value: draft wins if present, else server value
        const displayValue = drafts.has(ev.id)
          ? (drafts.get(ev.id) ?? "")
          : (cells.get(ev.id)?.value ?? "");
        const validationError = validationErrors.get(ev.id);
        const cellStatus = cellStatuses.get(ev.id) ?? "idle";

        return (
          <TableCell key={ev.id} className="min-w-[100px]">
            <div className="flex flex-col gap-1">
              <Input
                type="text"
                inputMode="decimal"
                value={displayValue}
                disabled={isWithdrawn || isSaving}
                onChange={(e) => handleChange(ev.id, e.target.value)}
                onBlur={(e) => handleBlur(ev.id, e.target.value)}
                className={cn(
                  "h-8 w-20 text-sm",
                  validationError && "border-destructive",
                  cellStatus === "saved" && "border-green-500",
                  cellStatus === "failed" && "border-destructive",
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
                {!validationError && cellStatus === "failed" && (
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
