import { toast } from "sonner";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { Section } from "@/gen/catalog/v1/catalog_pb";
import { useCreateSection } from "../hooks/useCreateSection";
import { useUpdateSection } from "../hooks/useUpdateSection";
import { type SectionFormValues } from "../schemas/section";
import { SectionForm, type SectionFormHelpers } from "./SectionForm";

interface SectionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** When provided the dialog operates in edit mode; absent = create mode. */
  section?: Section;
}

export function SectionDialog({
  open,
  onOpenChange,
  section,
}: SectionDialogProps) {
  const isEdit = section !== undefined;
  const createMutation = useCreateSection();
  const updateMutation = useUpdateSection();

  const handleSubmit = async (
    values: SectionFormValues,
    _helpers: SectionFormHelpers,
  ) => {
    try {
      if (isEdit) {
        await updateMutation.mutateAsync({
          id: section.id,
          seatCapacity: values.seatCapacity,
        });
        onOpenChange(false);
        toast.success("Sección actualizada");
      } else {
        await createMutation.mutateAsync({
          courseId: values.courseId,
          academicPeriodId: values.academicPeriodId,
          seatCapacity: values.seatCapacity,
        });
        onOpenChange(false);
        toast.success("Sección creada");
      }
    } catch {
      toast.error(
        isEdit
          ? "No se pudo actualizar la sección. Inténtalo de nuevo."
          : "No se pudo crear la sección. Inténtalo de nuevo.",
      );
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {isEdit ? "Editar sección" : "Crear sección"}
          </DialogTitle>
          <DialogDescription>
            {isEdit
              ? "Edita la capacidad de la sección."
              : "Completa los datos de la nueva sección."}
          </DialogDescription>
        </DialogHeader>
        <SectionForm
          onSubmit={handleSubmit}
          idPrefix="dialog-section"
          editMode={isEdit}
          defaultValues={
            isEdit
              ? {
                  courseId: section.courseId,
                  academicPeriodId: section.academicPeriodId,
                  seatCapacity: section.seatCapacity,
                }
              : undefined
          }
          submitLabel={isEdit ? "Guardar cambios" : "Crear sección"}
        />
      </DialogContent>
    </Dialog>
  );
}
