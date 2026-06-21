import { toast } from "sonner";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { AcademicPeriod } from "@/gen/catalog/v1/catalog_pb";
import { useCreateAcademicPeriod } from "../hooks/useCreateAcademicPeriod";
import { useUpdateAcademicPeriod } from "../hooks/useUpdateAcademicPeriod";
import { type AcademicPeriodFormValues } from "../schemas/academicPeriod";
import {
  AcademicPeriodForm,
  type AcademicPeriodFormHelpers,
} from "./AcademicPeriodForm";
import { mapAcademicPeriodMutationError } from "./errorMapping";

interface AcademicPeriodDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** When provided the dialog operates in edit mode; absent = create mode. */
  period?: AcademicPeriod;
}

export function AcademicPeriodDialog({
  open,
  onOpenChange,
  period,
}: AcademicPeriodDialogProps) {
  const isEdit = period !== undefined;
  const createMutation = useCreateAcademicPeriod();
  const updateMutation = useUpdateAcademicPeriod();

  const handleSubmit = async (
    values: AcademicPeriodFormValues,
    { setError }: AcademicPeriodFormHelpers,
  ) => {
    try {
      if (isEdit) {
        await updateMutation.mutateAsync({
          id: period.id,
          year: values.year,
          term: values.term,
          startDate: values.startDate,
          endDate: values.endDate,
        });
        onOpenChange(false);
        toast.success("Período actualizado");
      } else {
        await createMutation.mutateAsync({
          year: values.year,
          term: values.term,
          startDate: values.startDate,
          endDate: values.endDate,
        });
        onOpenChange(false);
        toast.success("Período creado");
      }
    } catch (err) {
      const result = mapAcademicPeriodMutationError(err);
      if (result === "handled-inline") {
        setError("year", {
          message: "Ya existe un período con ese año y semestre",
        });
      } else {
        toast.error(
          isEdit
            ? "No se pudo actualizar el período. Inténtalo de nuevo."
            : "No se pudo crear el período. Inténtalo de nuevo.",
        );
      }
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {isEdit ? "Editar período" : "Crear período"}
          </DialogTitle>
          <DialogDescription>
            {isEdit
              ? "Edita los datos del período académico."
              : "Completa los datos del nuevo período académico."}
          </DialogDescription>
        </DialogHeader>
        <AcademicPeriodForm
          onSubmit={handleSubmit}
          idPrefix="dialog-academic-period"
          defaultValues={
            isEdit
              ? {
                  year: period.year,
                  term: period.term,
                  startDate: period.startDate,
                  endDate: period.endDate,
                }
              : undefined
          }
          submitLabel={isEdit ? "Guardar cambios" : "Crear período"}
        />
      </DialogContent>
    </Dialog>
  );
}
