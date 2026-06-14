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
import type { Program } from "@/gen/catalog/v1/catalog_pb";
import { useDeleteProgram } from "../hooks/useDeleteProgram";
import { mapDeleteError } from "./errorMapping";

interface DeleteProgramDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  program: Program;
}

export function DeleteProgramDialog({
  open,
  onOpenChange,
  program,
}: DeleteProgramDialogProps) {
  const deleteMutation = useDeleteProgram();
  const [inlineError, setInlineError] = useState<
    "precondition" | "transport" | null
  >(null);

  const handleConfirm = async (e: React.MouseEvent) => {
    // Prevent AlertDialog from closing automatically on action click.
    e.preventDefault();
    setInlineError(null);
    try {
      await deleteMutation.mutateAsync({ id: program.id });
      onOpenChange(false);
      toast.success("Carrera eliminada");
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
          <AlertDialogTitle>¿Eliminar carrera?</AlertDialogTitle>
          <AlertDialogDescription>
            ¿Eliminar la carrera {program.code}? Esta acción no se puede
            deshacer.
          </AlertDialogDescription>
        </AlertDialogHeader>

        {inlineError === "precondition" && (
          <p role="alert" className="text-destructive text-sm">
            No se puede eliminar: la carrera tiene asignaturas o cupos
            asociados. Quita esas asociaciones primero.
          </p>
        )}

        {inlineError === "transport" && (
          <p role="alert" className="text-destructive text-sm">
            No se pudo eliminar la carrera. Inténtalo de nuevo.
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
