import { zodResolver } from "@hookform/resolvers/zod";
import { LoaderCircle, Save } from "lucide-react";
import type { UseFormSetError } from "react-hook-form";
import { useForm } from "react-hook-form";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  type ProgramQuotaFormValues,
  programQuotaSchema,
} from "../schemas/programQuota";

export interface ProgramQuotaFormHelpers {
  setError: UseFormSetError<ProgramQuotaFormValues>;
}

interface ProgramQuotaFormProps {
  onSubmit: (
    values: ProgramQuotaFormValues,
    helpers: ProgramQuotaFormHelpers,
  ) => Promise<void> | void;
  defaultValues?: Partial<ProgramQuotaFormValues>;
  submitLabel?: string;
  /** Optional id prefix for label/input pairs. Defaults to "program-quota". */
  idPrefix?: string;
}

export function ProgramQuotaForm({
  onSubmit,
  defaultValues,
  submitLabel = "Guardar",
  idPrefix = "program-quota",
}: ProgramQuotaFormProps) {
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<ProgramQuotaFormValues>({
    resolver: zodResolver(programQuotaSchema),
    mode: "onBlur",
    defaultValues: {
      year: undefined as unknown as number,
      admissionQuota: undefined as unknown as number,
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
          min={2000}
          max={2100}
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
        <Label htmlFor={`${idPrefix}-admissionQuota`}>Cupo</Label>
        <Input
          id={`${idPrefix}-admissionQuota`}
          type="number"
          min={1}
          max={10000}
          placeholder="ej. 60"
          aria-invalid={errors.admissionQuota ? true : undefined}
          {...register("admissionQuota", { valueAsNumber: true })}
        />
        <div className="min-h-[1.25rem]">
          {errors.admissionQuota && (
            <p role="alert" className="text-destructive text-sm">
              {errors.admissionQuota.message}
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
