import { toast } from "sonner";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { Program } from "@/gen/catalog/v1/catalog_pb";
import { useCreateProgram } from "../hooks/useCreateProgram";
import { useUpdateProgram } from "../hooks/useUpdateProgram";
import { type ProgramFormValues } from "../schemas/program";
import { mapProgramMutationError } from "./errorMapping";
import { ProgramForm, type ProgramFormHelpers } from "./ProgramForm";

interface ProgramDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** When provided the dialog operates in edit mode; absent = create mode. */
  program?: Program;
}

export function ProgramDialog({
  open,
  onOpenChange,
  program,
}: ProgramDialogProps) {
  const isEdit = program !== undefined;
  const createMutation = useCreateProgram();
  const updateMutation = useUpdateProgram();

  const handleSubmit = async (
    values: ProgramFormValues,
    { setError }: ProgramFormHelpers,
  ) => {
    try {
      if (isEdit) {
        await updateMutation.mutateAsync({
          id: program.id,
          code: values.code,
          name: values.name,
        });
        onOpenChange(false);
        toast.success("Programa actualizado");
      } else {
        await createMutation.mutateAsync({
          code: values.code,
          name: values.name,
        });
        onOpenChange(false);
        toast.success("Programa creado");
      }
    } catch (err) {
      const result = mapProgramMutationError(err, setError);
      if (result === "toast") {
        toast.error(
          isEdit
            ? "No se pudo actualizar el programa. Inténtalo de nuevo."
            : "No se pudo crear el programa. Inténtalo de nuevo.",
        );
      }
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {isEdit ? "Editar programa" : "Crear programa"}
          </DialogTitle>
        </DialogHeader>
        <ProgramForm
          onSubmit={handleSubmit}
          idPrefix="dialog-program"
          defaultValues={
            isEdit ? { code: program.code, name: program.name } : undefined
          }
          submitLabel={isEdit ? "Guardar cambios" : "Crear programa"}
        />
      </DialogContent>
    </Dialog>
  );
}
