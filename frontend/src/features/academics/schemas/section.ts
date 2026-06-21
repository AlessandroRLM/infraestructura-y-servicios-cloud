import { z } from "zod";

export const sectionSchema = z.object({
  courseId: z.string().min(1, { error: "Selecciona una asignatura" }),
  academicPeriodId: z
    .string()
    .min(1, { error: "Selecciona un período académico" }),
  seatCapacity: z
    .number({ error: "Ingresa la capacidad" })
    .int({ error: "La capacidad debe ser un número entero" })
    .min(1, { error: "La capacidad mínima es 1" })
    .max(500, { error: "La capacidad máxima es 500" }),
});

export type SectionFormValues = z.infer<typeof sectionSchema>;
