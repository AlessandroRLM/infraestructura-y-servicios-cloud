import { zodResolver } from "@hookform/resolvers/zod";
import { LoaderCircle, Save } from "lucide-react";
import type { UseFormSetError } from "react-hook-form";
import { useForm } from "react-hook-form";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { type ProgramFormValues, programSchema } from "../schemas/program";

export interface ProgramFormHelpers {
  setError: UseFormSetError<ProgramFormValues>;
}

interface ProgramFormProps {
  onSubmit: (
    values: ProgramFormValues,
    helpers: ProgramFormHelpers,
  ) => Promise<void> | void;
  defaultValues?: Partial<ProgramFormValues>;
  submitLabel?: string;
  /** Optional id prefix for label/input pairs. Defaults to "program". */
  idPrefix?: string;
}

export function ProgramForm({
  onSubmit,
  defaultValues,
  submitLabel = "Guardar",
  idPrefix = "program",
}: ProgramFormProps) {
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<ProgramFormValues>({
    resolver: zodResolver(programSchema),
    mode: "onBlur",
    defaultValues: { code: "", name: "", ...defaultValues },
  });

  return (
    <form
      noValidate
      onSubmit={handleSubmit((values) => onSubmit(values, { setError }))}
      className="space-y-4"
    >
      <div className="space-y-2">
        <Label htmlFor={`${idPrefix}-code`}>Código</Label>
        <Input
          id={`${idPrefix}-code`}
          placeholder="ej. ING-01"
          aria-invalid={errors.code ? true : undefined}
          {...register("code")}
        />
        <div className="min-h-[1.25rem]">
          {errors.code && (
            <p role="alert" className="text-destructive text-sm">
              {errors.code.message}
            </p>
          )}
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor={`${idPrefix}-name`}>Nombre</Label>
        <Input
          id={`${idPrefix}-name`}
          placeholder="ej. Ingeniería de Software"
          aria-invalid={errors.name ? true : undefined}
          {...register("name")}
        />
        <div className="min-h-[1.25rem]">
          {errors.name && (
            <p role="alert" className="text-destructive text-sm">
              {errors.name.message}
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
