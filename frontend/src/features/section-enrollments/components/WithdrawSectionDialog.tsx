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
import type { SectionEnrollment } from "@/gen/section_enrollment/v1/section_enrollment_pb";
import { mapWithdrawSectionError } from "../hooks/errorMapping";
import { useDisplayNames } from "../hooks/useDisplayNames";
import { useWithdrawSection } from "../hooks/useWithdrawSection";

interface WithdrawSectionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  sectionEnrollment: SectionEnrollment;
}

/**
 * AlertDialog confirmation for withdrawing a student from a section.
 * Mirrors CancelEnrollmentDialog pattern:
 *  - e.preventDefault() on action to prevent auto-close.
 *  - Inline error state for FailedPrecondition and transport errors.
 *  - Success: close + toast.success.
 *  - Cancel button label: "Volver".
 */
export function WithdrawSectionDialog({
  open,
  onOpenChange,
  sectionEnrollment,
}: WithdrawSectionDialogProps) {
  const withdrawMutation = useWithdrawSection();
  const [inlineError, setInlineError] = useState<
    "precondition" | "not_found" | "transport" | null
  >(null);

  // Resolve the student display name for the confirmation message.
  const nameMap = useDisplayNames([sectionEnrollment.studentId]);

  const handleConfirm = async (e: React.MouseEvent) => {
    e.preventDefault();
    setInlineError(null);
    try {
      await withdrawMutation.mutateAsync({ id: sectionEnrollment.id });
      onOpenChange(false);
      toast.success("Inscripción retirada");
    } catch (err) {
      const kind = mapWithdrawSectionError(err);
      setInlineError(kind);
    }
  };

  const handleOpenChange = (next: boolean) => {
    if (!next) setInlineError(null);
    onOpenChange(next);
  };

  const studentDisplay =
    (nameMap.get(sectionEnrollment.studentId) ??
      sectionEnrollment.studentId.slice(0, 8)) ||
    sectionEnrollment.enrollmentId.slice(0, 8);

  return (
    <AlertDialog open={open} onOpenChange={handleOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>¿Retirar inscripción?</AlertDialogTitle>
          <AlertDialogDescription>
            Esta acción retirará la inscripción del estudiante{" "}
            <strong>{studentDisplay}</strong> de esta sección. La inscripción
            quedará en estado retirado.
          </AlertDialogDescription>
        </AlertDialogHeader>

        {inlineError === "precondition" && (
          <p role="alert" className="text-destructive text-sm">
            No se puede retirar: la inscripción no está en estado activo.
          </p>
        )}

        {inlineError === "not_found" && (
          <p role="alert" className="text-destructive text-sm">
            No se encontró la inscripción. Es posible que ya haya sido retirada.
          </p>
        )}

        {inlineError === "transport" && (
          <p role="alert" className="text-destructive text-sm">
            No se pudo retirar la inscripción. Inténtalo de nuevo.
          </p>
        )}

        <AlertDialogFooter>
          <AlertDialogCancel>Volver</AlertDialogCancel>
          <AlertDialogAction
            onClick={handleConfirm}
            disabled={withdrawMutation.isPending}
            className="gap-2 bg-destructive hover:bg-destructive/90"
          >
            {withdrawMutation.isPending ? (
              <>
                <LoaderCircle className="size-4 animate-spin" aria-hidden />
                Retirando…
              </>
            ) : (
              "Retirar"
            )}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
