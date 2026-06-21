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
  academicPeriodLabel,
  useAcademicPeriods,
  useCourses,
} from "@/core/catalog";
import { type SectionFormValues, sectionSchema } from "../schemas/section";

export interface SectionFormHelpers {
  setError: UseFormSetError<SectionFormValues>;
}

interface SectionFormProps {
  onSubmit: (
    values: SectionFormValues,
    helpers: SectionFormHelpers,
  ) => Promise<void> | void;
  defaultValues?: Partial<SectionFormValues>;
  submitLabel?: string;
  /** When true, course and period selectors are read-only (edit mode only updates capacity). */
  editMode?: boolean;
  /** Optional id prefix for label/input pairs. Defaults to "section". */
  idPrefix?: string;
}

export function SectionForm({
  onSubmit,
  defaultValues,
  submitLabel = "Guardar",
  editMode = false,
  idPrefix = "section",
}: SectionFormProps) {
  const {
    register,
    control,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<SectionFormValues>({
    resolver: zodResolver(sectionSchema),
    mode: "onBlur",
    defaultValues: {
      courseId: "",
      academicPeriodId: "",
      seatCapacity: undefined as unknown as number,
      ...defaultValues,
    },
  });

  const { courses } = useCourses();
  const { periods } = useAcademicPeriods();

  return (
    <form
      noValidate
      onSubmit={handleSubmit((values) => onSubmit(values, { setError }))}
      className="space-y-4"
    >
      <div className="space-y-2">
        <Label htmlFor={`${idPrefix}-courseId`}>Asignatura</Label>
        <Controller
          name="courseId"
          control={control}
          render={({ field }) => (
            <Select
              value={field.value}
              onValueChange={field.onChange}
              disabled={editMode}
            >
              <SelectTrigger
                id={`${idPrefix}-courseId`}
                aria-invalid={errors.courseId ? true : undefined}
                onBlur={field.onBlur}
              >
                <SelectValue placeholder="Selecciona una asignatura" />
              </SelectTrigger>
              <SelectContent>
                {courses.map((c) => (
                  <SelectItem key={c.id} value={c.id}>
                    {c.code} — {c.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
        />
        <div className="min-h-[1.25rem]">
          {errors.courseId && (
            <p role="alert" className="text-destructive text-sm">
              {errors.courseId.message}
            </p>
          )}
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor={`${idPrefix}-academicPeriodId`}>
          Período académico
        </Label>
        <Controller
          name="academicPeriodId"
          control={control}
          render={({ field }) => (
            <Select
              value={field.value}
              onValueChange={field.onChange}
              disabled={editMode}
            >
              <SelectTrigger
                id={`${idPrefix}-academicPeriodId`}
                aria-invalid={errors.academicPeriodId ? true : undefined}
                onBlur={field.onBlur}
              >
                <SelectValue placeholder="Selecciona un período" />
              </SelectTrigger>
              <SelectContent>
                {periods.map((p) => (
                  <SelectItem key={p.id} value={p.id}>
                    {academicPeriodLabel(p.year, p.term)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
        />
        <div className="min-h-[1.25rem]">
          {errors.academicPeriodId && (
            <p role="alert" className="text-destructive text-sm">
              {errors.academicPeriodId.message}
            </p>
          )}
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor={`${idPrefix}-seatCapacity`}>Capacidad</Label>
        <Input
          id={`${idPrefix}-seatCapacity`}
          type="number"
          min={1}
          max={500}
          placeholder="ej. 30"
          aria-invalid={errors.seatCapacity ? true : undefined}
          {...register("seatCapacity", { valueAsNumber: true })}
        />
        <div className="min-h-[1.25rem]">
          {errors.seatCapacity && (
            <p role="alert" className="text-destructive text-sm">
              {errors.seatCapacity.message}
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
