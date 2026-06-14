import { toast } from "sonner";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { Course } from "@/gen/catalog/v1/catalog_pb";
import { useCreateCourse } from "../hooks/useCreateCourse";
import { useUpdateCourse } from "../hooks/useUpdateCourse";
import { type CourseFormValues } from "../schemas/course";
import { CourseForm, type CourseFormHelpers } from "./CourseForm";
import { mapCourseMutationError } from "./errorMapping";

interface CourseDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** When provided the dialog operates in edit mode; absent = create mode. */
  course?: Course;
}

export function CourseDialog({
  open,
  onOpenChange,
  course,
}: CourseDialogProps) {
  const isEdit = course !== undefined;
  const createMutation = useCreateCourse();
  const updateMutation = useUpdateCourse();

  const handleSubmit = async (
    values: CourseFormValues,
    { setError }: CourseFormHelpers,
  ) => {
    try {
      if (isEdit) {
        await updateMutation.mutateAsync({
          id: course.id,
          code: values.code,
          name: values.name,
          credits: values.credits,
        });
        onOpenChange(false);
        toast.success("Asignatura actualizada");
      } else {
        await createMutation.mutateAsync({
          code: values.code,
          name: values.name,
          credits: values.credits,
        });
        onOpenChange(false);
        toast.success("Asignatura creada");
      }
    } catch (err) {
      const result = mapCourseMutationError(err, setError);
      if (result === "toast") {
        toast.error(
          isEdit
            ? "No se pudo actualizar la asignatura. Inténtalo de nuevo."
            : "No se pudo crear la asignatura. Inténtalo de nuevo.",
        );
      }
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {isEdit ? "Editar asignatura" : "Crear asignatura"}
          </DialogTitle>
          <DialogDescription>
            {isEdit
              ? "Edita los datos de la asignatura."
              : "Completa los datos de la nueva asignatura."}
          </DialogDescription>
        </DialogHeader>
        <CourseForm
          onSubmit={handleSubmit}
          idPrefix="dialog-course"
          defaultValues={
            isEdit
              ? {
                  code: course.code,
                  name: course.name,
                  credits: course.credits,
                }
              : undefined
          }
          submitLabel={isEdit ? "Guardar cambios" : "Crear asignatura"}
        />
      </DialogContent>
    </Dialog>
  );
}
