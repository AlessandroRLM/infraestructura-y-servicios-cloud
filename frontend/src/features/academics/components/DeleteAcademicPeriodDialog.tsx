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
import { academicPeriodLabel } from "@/core/catalog";
import type { AcademicPeriod } from "@/gen/catalog/v1/catalog_pb";
import { useDeleteAcademicPeriod } from "../hooks/useDeleteAcademicPeriod";
import { mapDeleteError } from "./errorMapping";

interface DeleteAcademicPeriodDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  period: AcademicPeriod;
}

export function DeleteAcademicPeriodDialog({
  open,
  onOpenChange,
  period,
}: DeleteAcademicPeriodDialogProps) {
  const deleteMutation = useDeleteAcademicPeriod();
  const [inlineError, setInlineError] = useState<
    "precondition" | "transport" | null
  >(null);

  const handleConfirm = async (e: React.MouseEvent) => {
    // Prevent AlertDialog from closing automatically on action click.
    e.preventDefault();
    setInlineError(null);
    try {
      await deleteMutation.mutateAsync({ id: period.id });
      onOpenChange(false);
      toast.success("Período eliminado");
    } catch (err) {
      const kind = mapDeleteError(err);
      setInlineError(kind);
    }
  };

  const handleOpenChange = (next: boolean) => {
    if (!next) setInlineError(null);
    onOpenChange(next);
  };

  const label = academicPeriodLabel(period.year, period.term);

  return (
    <AlertDialog open={open} onOpenChange={handleOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>¿Eliminar período?</AlertDialogTitle>
          <AlertDialogDescription>
            ¿Eliminar el período {label}? Esta acción no se puede deshacer.
          </AlertDialogDescription>
        </AlertDialogHeader>

        {inlineError === "precondition" && (
          <p role="alert" className="text-destructive text-sm">
            No se puede eliminar: el período está en uso por secciones. Quita
            esas secciones primero.
          </p>
        )}

        {inlineError === "transport" && (
          <p role="alert" className="text-destructive text-sm">
            No se pudo eliminar el período. Inténtalo de nuevo.
          </p>
        )}

        <AlertDialogFooter>
          <AlertDialogCancel>Cancelar</AlertDialogCancel>
          <AlertDialogAction
            onClick={handleConfirm}
            disabled={deleteMutation.isPending}
            className="gap-2"
          >
            {deleteMutation.isPending ? (
              <>
                <LoaderCircle className="size-4 animate-spin" aria-hidden />
                Eliminando…
              </>
            ) : (
              "Eliminar"
            )}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
