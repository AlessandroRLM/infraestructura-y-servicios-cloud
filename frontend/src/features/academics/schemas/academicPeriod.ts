import { z } from "zod";

export const academicPeriodSchema = z
  .object({
    year: z
      .number({ error: "El año es obligatorio" })
      .int({ error: "El año debe ser un número entero" })
      .min(2000, { error: "El año debe ser 2000 o posterior" })
      .max(2100, { error: "El año no puede superar 2100" }),
    term: z
      .number({ error: "El semestre es obligatorio" })
      .int({ error: "El semestre debe ser un número entero" })
      .min(1, { error: "El semestre debe ser 1 o 2" })
      .max(2, { error: "El semestre debe ser 1 o 2" }),
    startDate: z
      .string()
      .min(1, { error: "La fecha de inicio es obligatoria" }),
    endDate: z.string().min(1, { error: "La fecha de término es obligatoria" }),
  })
  .refine((data) => data.endDate >= data.startDate, {
    error: "La fecha de término debe ser igual o posterior a la de inicio",
    path: ["endDate"],
  });

export type AcademicPeriodFormValues = z.infer<typeof academicPeriodSchema>;
