import { zodResolver } from "@hookform/resolvers/zod";
import { LoaderCircle, Minus, Plus, Save } from "lucide-react";
import { useRef, useState } from "react";
import {
  type Resolver,
  useFieldArray,
  useForm,
  useWatch,
} from "react-hook-form";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  type EvaluationSchemeFormValues,
  evaluationSchemeSchema,
} from "../schemas/evaluationScheme";
import { percentToWeight, sumPercents } from "../weights";
import type { SchemeErrorKind } from "./errorMapping";

/** Initial row shape used to seed the form. */
export interface InitialRow {
  percent: number;
}

interface EvaluationSchemeFormProps {
  /** "create" for a schemeless course; "recreate" for an existing scheme. */
  mode: "create" | "recreate";
  /**
   * Pre-filled rows for the editor.
   * For "recreate", these come from weightToPercent() on the current evaluations.
   *
   * NOTE (display-only round-trip): A hand-seeded weight like "0.333" will
   * display as 33% (Math.round(0.333*100)). If the admin submits without
   * editing, the saved value becomes "0.330". This is acceptable per design:
   * any scheme created via this UI already uses clean integer percents, and
   * the admin is explicitly recreating.
   */
  initialRows: InitialRow[];
  /**
   * Called on valid submit with the ordered weight strings (3-decimal, sum=1.000).
   * The caller is responsible for calling the appropriate RPC.
   */
  onSubmit: (weights: string[]) => Promise<void> | void;
  /** True while a mutation is in flight. */
  isSubmitting: boolean;
  /**
   * True once ListEvaluations has resolved (empty or non-empty).
   * False while loading or in error state — blocks submit regardless of total.
   */
  schemeStateKnown: boolean;
  /**
   * An error kind from mapSchemeError() to display in the form.
   * "precondition" → inline non-dismissable banner (editor stays open).
   * "already-exists" → inline message.
   * "generic" → handled by parent as toast; not shown here.
   * undefined | null → no error.
   */
  submitError?: SchemeErrorKind | null;
}

const PRECONDITION_MESSAGE =
  "Este curso ya tiene notas registradas. No es posible reemplazar el esquema.";
const ALREADY_EXISTS_MESSAGE =
  "El esquema ya existe. Recarga la página e intenta de nuevo.";

/** Returns a fresh default row. The percent starts at 0 but zod requires ≥1; the user must fill it. */
function emptyRow(): EvaluationSchemeFormValues["rows"][number] {
  return { percent: 0 };
}

/**
 * Weight editor form: react-hook-form + zodResolver + useFieldArray.
 * Rows hold integer percent values (1..100). Live running total is computed
 * via useWatch + sumPercents. Submit gate blocks when total !== 100,
 * isSubmitting, or !schemeStateKnown.
 *
 * In "recreate" mode, wraps submit in an AlertDialog confirmation because
 * recreating a scheme is a destructive operation.
 */
export function EvaluationSchemeForm({
  mode,
  initialRows,
  onSubmit,
  isSubmitting,
  schemeStateKnown,
  submitError,
}: EvaluationSchemeFormProps) {
  const [confirmOpen, setConfirmOpen] = useState(false);
  // Carries validated form values across the render boundary to the AlertDialog confirm handler.
  const pendingValuesRef = useRef<EvaluationSchemeFormValues | null>(null);

  // zodResolver infers the INPUT type from z.coerce (unknown), but the form
  // state uses the OUTPUT type (percent: number). Cast to align the types.
  const resolver = zodResolver(
    evaluationSchemeSchema,
  ) as unknown as Resolver<EvaluationSchemeFormValues>;

  const form = useForm<EvaluationSchemeFormValues>({
    resolver,
    mode: "onBlur",
    defaultValues: {
      rows: initialRows.length > 0 ? initialRows : [emptyRow()],
    },
  });

  const { fields, append, remove } = useFieldArray({
    control: form.control,
    name: "rows",
  });

  // Live total — display only, no submit side-effect. useWatch re-renders on every change.
  const watchedRows = useWatch({ control: form.control, name: "rows" });
  const parsedRows = (watchedRows ?? []).map((r) => {
    const n = Number(r.percent);
    return { percent: n };
  });
  const total = sumPercents(
    parsedRows.map((r) => ({
      percent: Number.isFinite(r.percent) ? r.percent : 0,
    })),
  );
  // Any non-integer value means zod .int() will reject on submit — disable the button.
  const hasNonInteger = parsedRows.some(
    (r) => !Number.isInteger(r.percent) || !Number.isFinite(r.percent),
  );

  const submitDisabled =
    total !== 100 || hasNonInteger || isSubmitting || !schemeStateKnown;

  const executeSubmit = async (values: EvaluationSchemeFormValues) => {
    const weights = values.rows.map((r) => percentToWeight(r.percent));
    try {
      await onSubmit(weights);
    } catch {
      // onSubmit rejections are handled by the parent (SchemeManagementView.handleSubmit).
      // Catching here prevents an unhandled rejection when the async chain is void-cast.
    }
  };

  const handleFormSubmit = form.handleSubmit((values) => {
    if (mode === "recreate") {
      pendingValuesRef.current = values;
      setConfirmOpen(true);
    } else {
      void executeSubmit(values);
    }
  });

  const handleConfirmRecreate = async () => {
    setConfirmOpen(false);
    const values = pendingValuesRef.current;
    if (values) {
      pendingValuesRef.current = null;
      await executeSubmit(values);
    }
  };

  return (
    <>
      <form
        noValidate
        onSubmit={handleFormSubmit}
        className="flex flex-col gap-4"
      >
        {/* Inline error banner for precondition / already-exists */}
        {submitError === "precondition" && (
          <div
            role="alert"
            className="rounded-md border border-destructive/50 bg-destructive/10 px-4 py-3"
          >
            <p className="text-destructive text-sm font-medium">
              {PRECONDITION_MESSAGE}
            </p>
          </div>
        )}
        {submitError === "already-exists" && (
          <div
            role="alert"
            className="rounded-md border border-destructive/50 bg-destructive/10 px-4 py-3"
          >
            <p className="text-destructive text-sm font-medium">
              {ALREADY_EXISTS_MESSAGE}
            </p>
          </div>
        )}

        {/* Evaluation rows */}
        <div className="flex flex-col gap-3">
          {fields.map((field, index) => (
            <div key={field.id} className="flex flex-col gap-1">
              <div className="flex items-center gap-2">
                <Label
                  htmlFor={`row-${index}-percent`}
                  className="w-32 shrink-0 text-sm"
                >
                  Evaluación {index + 1}
                </Label>
                <div className="flex items-center gap-1">
                  <Input
                    id={`row-${index}-percent`}
                    type="number"
                    inputMode="numeric"
                    min={1}
                    max={100}
                    step={1}
                    placeholder="ej. 30"
                    aria-invalid={
                      form.formState.errors.rows?.[index]?.percent
                        ? true
                        : undefined
                    }
                    className="w-24"
                    {...form.register(`rows.${index}.percent`)}
                  />
                  <span className="text-sm text-muted-foreground">%</span>
                </div>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  aria-label={`Eliminar Evaluación ${index + 1}`}
                  disabled={fields.length === 1 || isSubmitting}
                  onClick={() => remove(index)}
                >
                  <Minus className="size-4" aria-hidden />
                </Button>
              </div>
              {/* Reserved error slot — always rendered to prevent CLS */}
              <div className="min-h-5 pl-[8.5rem]">
                {form.formState.errors.rows?.[index]?.percent && (
                  <p role="alert" className="text-destructive text-sm">
                    {form.formState.errors.rows[index].percent.message}
                  </p>
                )}
              </div>
            </div>
          ))}
        </div>

        {/* Add row */}
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={isSubmitting}
          onClick={() => append(emptyRow())}
          className="gap-2 self-start"
        >
          <Plus className="size-4" aria-hidden />
          Agregar evaluación
        </Button>

        {/* Live total */}
        <div className="flex items-center justify-between rounded-md border px-3 py-2 text-sm">
          <span className="text-muted-foreground">Total</span>
          <span
            className={
              total === 100 ? "font-medium" : "font-medium text-destructive"
            }
          >
            {total}% / 100%
          </span>
        </div>

        {/* Submit */}
        <Button type="submit" disabled={submitDisabled} className="gap-2">
          {isSubmitting ? (
            <>
              <LoaderCircle className="size-4 animate-spin" aria-hidden />
              Guardando…
            </>
          ) : (
            <>
              <Save className="size-4" aria-hidden />
              {mode === "recreate" ? "Recrear esquema" : "Crear esquema"}
            </>
          )}
        </Button>
      </form>

      {/* Confirm dialog — only shown in recreate mode */}
      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>¿Recrear esquema?</AlertDialogTitle>
            <AlertDialogDescription>
              Esta acción reemplazará el esquema de evaluación actual. No se
              puede deshacer si no hay notas registradas.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isSubmitting}>
              Cancelar
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={isSubmitting}
              onClick={() => void handleConfirmRecreate()}
            >
              {isSubmitting ? (
                <>
                  <LoaderCircle className="size-4 animate-spin" aria-hidden />
                  Guardando…
                </>
              ) : (
                "Recrear"
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
