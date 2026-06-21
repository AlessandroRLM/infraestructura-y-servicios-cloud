import { toast } from "sonner";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { ProgramQuota } from "@/gen/catalog/v1/catalog_pb";
import { useCreateProgramQuota } from "../hooks/useCreateProgramQuota";
import { useUpdateProgramQuota } from "../hooks/useUpdateProgramQuota";
import { type ProgramQuotaFormValues } from "../schemas/programQuota";
import {
  ProgramQuotaForm,
  type ProgramQuotaFormHelpers,
} from "./ProgramQuotaForm";

interface ProgramQuotaDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** UUID of the program this quota belongs to. Passed into create; immutable on edit. */
  programId: string;
  /** When provided the dialog operates in edit mode; absent = create mode. */
  quota?: ProgramQuota;
}

export function ProgramQuotaDialog({
  open,
  onOpenChange,
  programId,
  quota,
}: ProgramQuotaDialogProps) {
  const isEdit = quota !== undefined;
  const createMutation = useCreateProgramQuota();
  const updateMutation = useUpdateProgramQuota();

  const handleSubmit = async (
    values: ProgramQuotaFormValues,
    _helpers: ProgramQuotaFormHelpers,
  ) => {
    try {
      if (isEdit) {
        await updateMutation.mutateAsync({
          id: quota.id,
          year: values.year,
          admissionQuota: values.admissionQuota,
        });
        onOpenChange(false);
        toast.success("Cupo actualizado");
      } else {
        await createMutation.mutateAsync({
          programId,
          year: values.year,
          admissionQuota: values.admissionQuota,
        });
        onOpenChange(false);
        toast.success("Cupo creado");
      }
    } catch {
      toast.error(
        isEdit
          ? "No se pudo actualizar el cupo. Inténtalo de nuevo."
          : "No se pudo crear el cupo. Inténtalo de nuevo.",
      );
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEdit ? "Editar cupo" : "Crear cupo"}</DialogTitle>
          <DialogDescription>
            {isEdit
              ? "Edita el año y el cupo de admisión."
              : "Completa los datos del nuevo cupo de admisión."}
          </DialogDescription>
        </DialogHeader>
        <ProgramQuotaForm
          onSubmit={handleSubmit}
          idPrefix="dialog-quota"
          defaultValues={
            isEdit
              ? {
                  year: quota.year,
                  admissionQuota: quota.admissionQuota,
                }
              : undefined
          }
          submitLabel={isEdit ? "Guardar cambios" : "Crear cupo"}
        />
      </DialogContent>
    </Dialog>
  );
}
