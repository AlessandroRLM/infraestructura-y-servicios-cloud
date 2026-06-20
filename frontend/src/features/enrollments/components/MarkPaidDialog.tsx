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
import { useMarkEnrollmentPaid } from "../hooks/useMarkEnrollmentPaid";
import { mapLifecycleError } from "../hooks/errorMapping";

interface MarkPaidDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  enrollment: Enrollment;
}

/**
 * AlertDialog confirmation for marking a pending enrollment as paid.
 * Mirrors DeleteCourseDialog pattern:
 *  - e.preventDefault() on action to prevent auto-close.
 *  - Inline error state for FailedPrecondition (precondition) and transport errors.
 *  - Success: close + toast.success.
 */
export function MarkPaidDialog({
  open,
  onOpenChange,
  enrollment,
}: MarkPaidDialogProps) {
  const markPaidMutation = useMarkEnrollmentPaid();
  const [inlineError, setInlineError] = useState<
    "precondition" | "transport" | null
  >(null);

  const handleConfirm = async (e: React.MouseEvent) => {
    e.preventDefault();
    setInlineError(null);
    try {
      await markPaidMutation.mutateAsync({ id: enrollment.id });
      onOpenChange(false);
      toast.success("Matrícula marcada como pagada");
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
          <AlertDialogTitle>
            ¿Marcar matrícula como pagada?
          </AlertDialogTitle>
          <AlertDialogDescription>
            Esta acción cambiará el estado de la matrícula de{" "}
            {enrollment.studentName || enrollment.studentId.slice(0, 8)} a{" "}
            <strong>pagada</strong>. Solo se puede realizar si la matrícula
            está en estado pendiente.
          </AlertDialogDescription>
        </AlertDialogHeader>

        {inlineError === "precondition" && (
          <p role="alert" className="text-destructive text-sm">
            No se puede marcar como pagada: la matrícula no está en estado
            pendiente.
          </p>
        )}

        {inlineError === "transport" && (
          <p role="alert" className="text-destructive text-sm">
            No se pudo completar la acción. Inténtalo de nuevo.
          </p>
        )}

        <AlertDialogFooter>
          <AlertDialogCancel>Cancelar</AlertDialogCancel>
          <AlertDialogAction
            onClick={handleConfirm}
            disabled={markPaidMutation.isPending}
            className="gap-2"
          >
            {markPaidMutation.isPending ? (
              <>
                <LoaderCircle className="size-4 animate-spin" aria-hidden />
                Marcando…
              </>
            ) : (
              "Marcar pagada"
            )}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
