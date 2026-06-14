import { zodResolver } from "@hookform/resolvers/zod";
import { LoaderCircle, Save } from "lucide-react";
import type { UseFormSetError } from "react-hook-form";
import { Controller, useForm } from "react-hook-form";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  type CourseFormValues,
  CREDIT_OPTIONS,
  courseSchema,
} from "../schemas/course";

export interface CourseFormHelpers {
  setError: UseFormSetError<CourseFormValues>;
}

interface CourseFormProps {
  onSubmit: (
    values: CourseFormValues,
    helpers: CourseFormHelpers,
  ) => Promise<void> | void;
  defaultValues?: Partial<CourseFormValues>;
  submitLabel?: string;
  /** Optional id prefix for label/input pairs. Defaults to "course". */
  idPrefix?: string;
}

export function CourseForm({
  onSubmit,
  defaultValues,
  submitLabel = "Guardar",
  idPrefix = "course",
}: CourseFormProps) {
  const {
    register,
    control,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<CourseFormValues>({
    resolver: zodResolver(courseSchema),
    mode: "onBlur",
    defaultValues: {
      code: "",
      name: "",
      credits: undefined as unknown as number,
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
        <Label htmlFor={`${idPrefix}-code`}>Código</Label>
        <Input
          id={`${idPrefix}-code`}
          placeholder="ej. CS-101"
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
          placeholder="ej. Cálculo Diferencial"
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

      <div className="space-y-2">
        <Label htmlFor={`${idPrefix}-credits`}>Créditos</Label>
        <Controller
          name="credits"
          control={control}
          render={({ field }) => (
            <Select
              value={field.value != null ? String(field.value) : ""}
              onValueChange={(v) => field.onChange(Number(v))}
            >
              <SelectTrigger
                id={`${idPrefix}-credits`}
                aria-invalid={errors.credits ? true : undefined}
                onBlur={field.onBlur}
              >
                <SelectValue placeholder="Selecciona créditos" />
              </SelectTrigger>
              <SelectContent>
                {CREDIT_OPTIONS.map((c) => (
                  <SelectItem key={c} value={String(c)}>
                    {c}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
        />
        <div className="min-h-[1.25rem]">
          {errors.credits && (
            <p role="alert" className="text-destructive text-sm">
              {errors.credits.message}
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
