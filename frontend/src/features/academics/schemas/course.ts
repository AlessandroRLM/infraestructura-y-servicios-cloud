import { z } from "zod";

export const CREDIT_OPTIONS = [3, 4, 5, 6, 7, 8, 9, 10, 11, 12] as const;

export const courseSchema = z.object({
  code: z.string().min(1, { error: "El código es obligatorio" }),
  name: z.string().min(1, { error: "El nombre es obligatorio" }),
  credits: z
    .number({ error: "Selecciona los créditos" })
    .refine((n) => (CREDIT_OPTIONS as readonly number[]).includes(n), {
      error: "Créditos fuera de rango (3–12)",
    }),
});

export type CourseFormValues = z.infer<typeof courseSchema>;
