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
import type { ProgramQuota } from "@/gen/catalog/v1/catalog_pb";
import { useDeleteProgramQuota } from "../hooks/useDeleteProgramQuota";
import { mapDeleteError } from "./errorMapping";

interface DeleteProgramQuotaDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  quota: ProgramQuota;
}

export function DeleteProgramQuotaDialog({
  open,
  onOpenChange,
  quota,
}: DeleteProgramQuotaDialogProps) {
  const deleteMutation = useDeleteProgramQuota();
  const [inlineError, setInlineError] = useState<
    "precondition" | "transport" | null
  >(null);

  const handleConfirm = async (e: React.MouseEvent) => {
    // Prevent AlertDialog from closing automatically on action click.
    e.preventDefault();
    setInlineError(null);
    try {
      await deleteMutation.mutateAsync({ id: quota.id });
      onOpenChange(false);
      toast.success("Cupo eliminado");
    } catch (err) {
      const kind = mapDeleteError(err);
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
          <AlertDialogTitle>¿Eliminar cupo?</AlertDialogTitle>
          <AlertDialogDescription>
            ¿Eliminar el cupo de {quota.admissionQuota} para el año {quota.year}
            ? Esta acción no se puede deshacer.
          </AlertDialogDescription>
        </AlertDialogHeader>

        {inlineError === "precondition" && (
          <p role="alert" className="text-destructive text-sm">
            No se puede eliminar: el cupo está en uso por inscripciones activas.
            Quita esas relaciones primero.
          </p>
        )}

        {inlineError === "transport" && (
          <p role="alert" className="text-destructive text-sm">
            No se pudo eliminar el cupo. Inténtalo de nuevo.
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
