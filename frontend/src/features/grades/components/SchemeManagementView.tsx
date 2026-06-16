import { RefreshCw } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useCreateEvaluationScheme } from "../hooks/useCreateEvaluationScheme";
import { useEvaluations } from "../hooks/useEvaluations";
import { useRecreateEvaluationScheme } from "../hooks/useRecreateEvaluationScheme";
import { weightToPercent } from "../weights";
import { CourseSchemePicker } from "./CourseSchemePicker";
import { CurrentSchemeDisplay } from "./CurrentSchemeDisplay";
import { EvaluationSchemeForm, type InitialRow } from "./EvaluationSchemeForm";
import { mapSchemeError, type SchemeErrorKind } from "./errorMapping";

/**
 * Container for the admin evaluation-scheme management UI.
 *
 * Owns the selected course ID state and orchestrates:
 * - CourseSchemePicker: searchable course selector
 * - useEvaluations: queries the current scheme for the selected course
 * - CurrentSchemeDisplay: shows existing evaluations or empty state
 * - EvaluationSchemeForm: weight editor, opened by create/recreate affordances
 * - useCreateEvaluationScheme / useRecreateEvaluationScheme: mutations
 * - mapSchemeError: maps RPC errors to user-facing messages
 */
export function SchemeManagementView() {
  const [courseId, setCourseId] = useState("");
  const [showForm, setShowForm] = useState(false);
  const [submitError, setSubmitError] = useState<SchemeErrorKind | null>(null);
  // Snapshotted at form-open time so a background refetch cannot flip create↔recreate
  // while the form is open.
  const [snapshotMode, setSnapshotMode] = useState<
    "create" | "recreate" | null
  >(null);

  const { evaluations, isPending, isError, refetch } = useEvaluations(courseId);

  const createMutation = useCreateEvaluationScheme(courseId);
  const recreateMutation = useRecreateEvaluationScheme(courseId);

  const isSubmitting = createMutation.isPending || recreateMutation.isPending;

  // Scheme state is known once a course is selected and the query has resolved
  // (either empty or non-empty). Prevents submitting before knowing create vs recreate.
  const schemeStateKnown = courseId !== "" && !isPending && !isError;

  const derivedMode = evaluations.length > 0 ? "recreate" : "create";
  // Use the snapshotted mode while the form is open; fall back to derived for display logic.
  const mode = snapshotMode ?? derivedMode;

  // Pre-fill form rows from the current scheme for recreate mode.
  const initialRows: InitialRow[] =
    mode === "recreate"
      ? evaluations
          .slice()
          .sort((a, b) => a.position - b.position)
          .map((ev) => ({ percent: weightToPercent(ev.weight) }))
      : [];

  const handleCourseChange = (newCourseId: string) => {
    setCourseId(newCourseId);
    setSubmitError(null);
    setShowForm(false);
    setSnapshotMode(null);
  };

  const handleCreateScheme = () => {
    setSnapshotMode("create");
    setShowForm(true);
  };

  const handleRecreateScheme = () => {
    setSnapshotMode("recreate");
    setShowForm(true);
  };

  const handleSubmit = async (weights: string[]) => {
    try {
      const evaluationInputs = weights.map((weight) => ({ weight }));
      if (mode === "create") {
        await createMutation.mutateAsync({
          courseId,
          evaluations: evaluationInputs,
        });
      } else {
        await recreateMutation.mutateAsync({
          courseId,
          evaluations: evaluationInputs,
        });
      }
      setShowForm(false);
      setSubmitError(null);
      setSnapshotMode(null);
      toast.success(
        mode === "create"
          ? "Esquema creado correctamente."
          : "Esquema recreado correctamente.",
      );
    } catch (err) {
      const kind = mapSchemeError(err);
      if (kind === "generic") {
        toast.error("No se pudo guardar el esquema. Inténtalo de nuevo.");
      } else {
        setSubmitError(kind);
      }
    }
  };

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-1">
        <h1 className="font-semibold text-2xl tracking-tight">Notas</h1>
        <p className="text-muted-foreground text-sm">
          Administra los esquemas de evaluación por asignatura.
        </p>
      </div>

      <div className="flex flex-col gap-4">
        <div className="flex flex-col gap-2">
          <span className="text-sm font-medium">Asignatura</span>
          <CourseSchemePicker value={courseId} onChange={handleCourseChange} />
        </div>

        {/* Scheme section — only shown when a course is selected */}
        {courseId !== "" && (
          <div className="flex flex-col gap-4">
            {isPending && (
              <div
                role="status"
                aria-busy="true"
                aria-label="Cargando esquema"
                className="flex flex-col gap-2"
              >
                {Array.from({ length: 3 }).map((_, i) => (
                  <Skeleton key={i} className="h-10 w-full" />
                ))}
              </div>
            )}

            {isError && (
              <div
                className="rounded-md border border-destructive/50 p-4"
                role="alert"
              >
                <p className="text-destructive text-sm font-medium">
                  No se pudo cargar el esquema. Intenta de nuevo.
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

            {!isPending && !isError && !showForm && (
              <CurrentSchemeDisplay
                evaluations={evaluations}
                onCreateScheme={handleCreateScheme}
                onRecreateScheme={handleRecreateScheme}
              />
            )}

            {!isPending && !isError && showForm && (
              <EvaluationSchemeForm
                mode={mode}
                initialRows={initialRows}
                onSubmit={handleSubmit}
                isSubmitting={isSubmitting}
                schemeStateKnown={schemeStateKnown}
                submitError={submitError}
              />
            )}
          </div>
        )}
      </div>
    </div>
  );
}
