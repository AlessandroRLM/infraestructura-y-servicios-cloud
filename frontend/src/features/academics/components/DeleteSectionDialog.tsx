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
import type { Section } from "@/gen/catalog/v1/catalog_pb";
import { useDeleteSection } from "../hooks/useDeleteSection";
import { mapDeleteError } from "./errorMapping";

interface DeleteSectionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  section: Section;
}

export function DeleteSectionDialog({
  open,
  onOpenChange,
  section,
}: DeleteSectionDialogProps) {
  const deleteMutation = useDeleteSection();
  const [inlineError, setInlineError] = useState<
    "precondition" | "transport" | null
  >(null);

  const handleConfirm = async (e: React.MouseEvent) => {
    // Prevent AlertDialog from closing automatically on action click.
    e.preventDefault();
    setInlineError(null);
    try {
      await deleteMutation.mutateAsync({ id: section.id });
      onOpenChange(false);
      toast.success("Sección eliminada");
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
          <AlertDialogTitle>¿Eliminar sección?</AlertDialogTitle>
          <AlertDialogDescription>
            ¿Eliminar la sección con capacidad {section.seatCapacity} asientos?
            Esta acción no se puede deshacer.
          </AlertDialogDescription>
        </AlertDialogHeader>

        {inlineError === "precondition" && (
          <p role="alert" className="text-destructive text-sm">
            No se puede eliminar: la sección está en uso por inscripciones u
            otras asociaciones. Quita esas relaciones primero.
          </p>
        )}

        {inlineError === "transport" && (
          <p role="alert" className="text-destructive text-sm">
            No se pudo eliminar la sección. Inténtalo de nuevo.
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
