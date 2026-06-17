import { z } from "zod";

/**
 * Schema for a single evaluation row in the weight editor.
 * Accepts a coerced integer percent in the range [1, 100].
 * The .int() constraint is the critical guard for the float-safety proof:
 * percentToWeight() is only exact for integer inputs.
 */
const rowSchema = z.object({
  percent: z.coerce
    .number({ error: "Ingresa un porcentaje" })
    .int({ error: "Usa números enteros" })
    .min(1, { error: "Mínimo 1%" })
    .max(100, { error: "Máximo 100%" }),
});

/**
 * Zod 4 schema for the evaluation scheme weight editor form.
 * Validates that all rows have integer percent values in [1, 100]
 * and that the total equals exactly 100.
 *
 * The cross-field refine (total === 100) is the submit gate contract:
 * zodResolver will mark the form invalid until the total is exactly 100.
 */
export const evaluationSchemeSchema = z
  .object({
    rows: z
      .array(rowSchema)
      .min(1, { error: "Agrega al menos una evaluación" }),
  })
  .refine((v) => v.rows.reduce((s, r) => s + r.percent, 0) === 100, {
    error: "Los porcentajes deben sumar exactamente 100%",
    path: ["rows"],
  });

/** Inferred form values type for the evaluation scheme editor. */
export type EvaluationSchemeFormValues = z.infer<typeof evaluationSchemeSchema>;
