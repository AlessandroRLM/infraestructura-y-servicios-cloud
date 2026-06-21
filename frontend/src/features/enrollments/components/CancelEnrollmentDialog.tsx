import { LoaderCircle } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
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
import type { Enrollment } from "@/gen/enrollment/v1/enrollment_pb";
import { mapLifecycleError } from "../hooks/errorMapping";
import { useCancelEnrollment } from "../hooks/useCancelEnrollment";

interface CancelEnrollmentDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  enrollment: Enrollment;
}

/**
 * AlertDialog confirmation for cancelling an enrollment.
 * Mirrors DeleteCourseDialog pattern:
 *  - e.preventDefault() on action to prevent auto-close.
 *  - Inline error state for FailedPrecondition (precondition) and transport errors.
 *  - Success: close + toast.success.
 *  - Cancel button label: "Volver" (per design §8).
 */
export function CancelEnrollmentDialog({
  open,
  onOpenChange,
  enrollment,
}: CancelEnrollmentDialogProps) {
  const cancelMutation = useCancelEnrollment();
  const [inlineError, setInlineError] = useState<
    "precondition" | "transport" | null
  >(null);

  const handleConfirm = async (e: React.MouseEvent) => {
    e.preventDefault();
    setInlineError(null);
    try {
      await cancelMutation.mutateAsync({ id: enrollment.id });
      onOpenChange(false);
      toast.success("Matrícula cancelada");
    } catch (err) {
      const kind = mapLifecycleError(err);
      setInlineError(kind);
    }
  };

  const handleOpenChange = (next: boolean) => {
    if (!next) setInlineError(null);
    onOpenChange(next);
  };

  return (
    <AlertDialog open={open} onOpenChange={handleOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>¿Cancelar matrícula?</AlertDialogTitle>
          <AlertDialogDescription>
            Esta acción cancelará la matrícula de{" "}
            {enrollment.studentName || enrollment.studentId.slice(0, 8)} en el
            programa{" "}
            {enrollment.programName || enrollment.programId.slice(0, 8)} (
            {enrollment.year}). Esta acción no se puede deshacer fácilmente.
          </AlertDialogDescription>
        </AlertDialogHeader>

        {inlineError === "precondition" && (
          <p role="alert" className="text-destructive text-sm">
            No se puede cancelar: la matrícula ya está cancelada o no cumple las
            condiciones requeridas.
          </p>
        )}

        {inlineError === "transport" && (
          <p role="alert" className="text-destructive text-sm">
            No se pudo cancelar la matrícula. Inténtalo de nuevo.
          </p>
        )}

        <AlertDialogFooter>
          <AlertDialogCancel>Volver</AlertDialogCancel>
          <AlertDialogAction
            onClick={handleConfirm}
            disabled={cancelMutation.isPending}
            className="gap-2"
          >
            {cancelMutation.isPending ? (
              <>
                <LoaderCircle className="size-4 animate-spin" aria-hidden />
                Cancelando…
              </>
            ) : (
              "Cancelar matrícula"
            )}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
