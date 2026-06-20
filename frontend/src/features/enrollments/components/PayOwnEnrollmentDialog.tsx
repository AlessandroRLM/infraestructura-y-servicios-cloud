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
import { useMarkOwnEnrollmentPaid } from "../hooks/useMarkOwnEnrollmentPaid";

interface PayOwnEnrollmentDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  enrollment: Enrollment;
}

/**
 * AlertDialog confirmation for a student paying their own pending enrollment.
 * Mirrors MarkPaidDialog:
 *  - e.preventDefault() on action to prevent auto-close.
 *  - Inline error state for FailedPrecondition (precondition) and transport errors.
 *  - Success: close + toast.success + cache invalidation via the hook.
 */
export function PayOwnEnrollmentDialog({
  open,
  onOpenChange,
  enrollment,
}: PayOwnEnrollmentDialogProps) {
  const markPaidMutation = useMarkOwnEnrollmentPaid();
  const [inlineError, setInlineError] = useState<
    "precondition" | "transport" | null
  >(null);

  const handleConfirm = async (e: React.MouseEvent) => {
    e.preventDefault();
    setInlineError(null);
    try {
      await markPaidMutation.mutateAsync({ id: enrollment.id });
      onOpenChange(false);
      toast.success("Matrícula pagada");
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
            ¿Confirmar pago de tu matrícula?
          </AlertDialogTitle>
          <AlertDialogDescription>
            Tu matrícula en{" "}
            <strong>{enrollment.programName}</strong> será marcada como{" "}
            <strong>pagada</strong>. Esta acción solo es posible si la
            matrícula está en estado pendiente.
          </AlertDialogDescription>
        </AlertDialogHeader>

        {inlineError === "precondition" && (
          <p role="alert" className="text-destructive text-sm">
            No se puede pagar en su estado actual.
          </p>
        )}

        {inlineError === "transport" && (
          <p role="alert" className="text-destructive text-sm">
            No se pudo procesar el pago. Inténtalo de nuevo.
          </p>
        )}

        <AlertDialogFooter>
          <AlertDialogCancel>Volver</AlertDialogCancel>
          <AlertDialogAction
            onClick={handleConfirm}
            disabled={markPaidMutation.isPending}
            className="gap-2"
          >
            {markPaidMutation.isPending ? (
              <>
                <LoaderCircle className="size-4 animate-spin" aria-hidden />
                Procesando…
              </>
            ) : (
              "Pagar"
            )}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
