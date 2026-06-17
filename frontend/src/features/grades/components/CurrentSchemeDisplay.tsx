import { ClipboardList, PlusCircle, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import type { Evaluation } from "@/gen/grades/v1/grades_pb";
import { weightToPercent } from "../weights";

interface CurrentSchemeDisplayProps {
  /** Evaluations for the selected course, ordered by position ascending. */
  evaluations: Evaluation[];
  /** Called when the user clicks "Crear esquema" in the empty state. */
  onCreateScheme: () => void;
  /** Called when the user clicks "Recrear esquema" in the non-empty state. */
  onRecreateScheme: () => void;
}

/**
 * Presentational component that displays the current evaluation scheme for a course.
 *
 * When evaluations is empty, renders an empty state with a "Crear esquema" affordance.
 * When evaluations is non-empty, renders an ordered list by position ascending,
 * each showing "Evaluación {position} — {percent}%", plus a running total and
 * a "Recrear esquema" affordance.
 */
export function CurrentSchemeDisplay({
  evaluations,
  onCreateScheme,
  onRecreateScheme,
}: CurrentSchemeDisplayProps) {
  if (evaluations.length === 0) {
    return (
      <Empty>
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <ClipboardList />
          </EmptyMedia>
          <EmptyTitle>Sin esquema</EmptyTitle>
          <EmptyDescription>
            Este curso no tiene un esquema todavía.
          </EmptyDescription>
        </EmptyHeader>
        <Button
          variant="outline"
          size="sm"
          onClick={onCreateScheme}
          className="gap-2"
        >
          <PlusCircle className="size-4" aria-hidden />
          Crear esquema
        </Button>
      </Empty>
    );
  }

  const sorted = [...evaluations].sort((a, b) => a.position - b.position);
  const total = sorted.reduce((acc, ev) => acc + weightToPercent(ev.weight), 0);

  return (
    <div className="flex flex-col gap-4">
      <ul className="flex flex-col gap-2">
        {sorted.map((ev) => (
          <li
            key={ev.id}
            className="flex items-center justify-between rounded-md border px-3 py-2 text-sm"
          >
            <span className="font-medium">Evaluación {ev.position}</span>
            <span className="text-muted-foreground">
              {weightToPercent(ev.weight)}%
            </span>
          </li>
        ))}
      </ul>

      <div className="flex items-center justify-between text-sm">
        <span className="text-muted-foreground">Total</span>
        <span className="font-medium">{total}%</span>
      </div>

      <Button
        variant="outline"
        size="sm"
        onClick={onRecreateScheme}
        className="gap-2 self-start"
      >
        <RefreshCw className="size-4" aria-hidden />
        Recrear esquema
      </Button>
    </div>
  );
}
