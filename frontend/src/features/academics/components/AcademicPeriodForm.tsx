import { zodResolver } from "@hookform/resolvers/zod";
import { LoaderCircle, Save } from "lucide-react";
import type { UseFormSetError } from "react-hook-form";
import { useForm } from "react-hook-form";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  type AcademicPeriodFormValues,
  academicPeriodSchema,
} from "../schemas/academicPeriod";

export interface AcademicPeriodFormHelpers {
  setError: UseFormSetError<AcademicPeriodFormValues>;
}

interface AcademicPeriodFormProps {
  onSubmit: (
    values: AcademicPeriodFormValues,
    helpers: AcademicPeriodFormHelpers,
  ) => Promise<void> | void;
  defaultValues?: Partial<AcademicPeriodFormValues>;
  submitLabel?: string;
  /** Optional id prefix for label/input pairs. Defaults to "academic-period". */
  idPrefix?: string;
}

export function AcademicPeriodForm({
  onSubmit,
  defaultValues,
  submitLabel = "Guardar",
  idPrefix = "academic-period",
}: AcademicPeriodFormProps) {
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<AcademicPeriodFormValues>({
    resolver: zodResolver(academicPeriodSchema),
    mode: "onBlur",
    defaultValues: {
      year: undefined as unknown as number,
      term: undefined as unknown as number,
      startDate: "",
      endDate: "",
      ...defaultValues,
    },
  });

  return (
    <form
      noValidate
      onSubmit={handleSubmit((values) => onSubmit(values, { setError }))}
      className="space-y-4"
    >
      <div className="space-y-2">
        <Label htmlFor={`${idPrefix}-year`}>Año</Label>
        <Input
          id={`${idPrefix}-year`}
          type="number"
          placeholder="ej. 2025"
          aria-invalid={errors.year ? true : undefined}
          {...register("year", { valueAsNumber: true })}
        />
        <div className="min-h-[1.25rem]">
          {errors.year && (
            <p role="alert" className="text-destructive text-sm">
              {errors.year.message}
            </p>
          )}
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor={`${idPrefix}-term`}>Semestre</Label>
        <Input
          id={`${idPrefix}-term`}
          type="number"
          placeholder="1 o 2"
          min={1}
          max={2}
          aria-invalid={errors.term ? true : undefined}
          {...register("term", { valueAsNumber: true })}
        />
        <div className="min-h-[1.25rem]">
          {errors.term && (
            <p role="alert" className="text-destructive text-sm">
              {errors.term.message}
            </p>
          )}
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor={`${idPrefix}-startDate`}>Inicio</Label>
        <Input
          id={`${idPrefix}-startDate`}
          type="date"
          aria-invalid={errors.startDate ? true : undefined}
          {...register("startDate")}
        />
        <div className="min-h-[1.25rem]">
          {errors.startDate && (
            <p role="alert" className="text-destructive text-sm">
              {errors.startDate.message}
            </p>
          )}
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor={`${idPrefix}-endDate`}>Término</Label>
        <Input
          id={`${idPrefix}-endDate`}
          type="date"
          aria-invalid={errors.endDate ? true : undefined}
          {...register("endDate")}
        />
        <div className="min-h-[1.25rem]">
          {errors.endDate && (
            <p role="alert" className="text-destructive text-sm">
              {errors.endDate.message}
            </p>
          )}
        </div>
      </div>

      <Button type="submit" className="w-full gap-2" disabled={isSubmitting}>
        {isSubmitting ? (
          <>
            <LoaderCircle className="size-4 animate-spin" aria-hidden />
            Guardando…
          </>
        ) : (
          <>
            <Save className="size-4" aria-hidden />
            {submitLabel}
          </>
        )}
      </Button>
    </form>
  );
}
