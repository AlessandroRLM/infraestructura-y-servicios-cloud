import { z } from "zod";

/**
 * Zod 4 schema for a Chilean grade input [1.0, 7.0] with at most one decimal place.
 * Accepts a raw string input (from an HTML input element) and coerces to number.
 * Rejects out-of-range values and more than one decimal digit.
 *
 * Used by GradeRow for per-cell blur validation before row save.
 */
export const gradeValueSchema = z.coerce
  .number({ error: "Ingresa un número entre 1.0 y 7.0" })
  .min(1.0, { error: "La nota mínima es 1.0" })
  .max(7.0, { error: "La nota máxima es 7.0" })
  .refine((v) => Math.round(v * 10) === v * 10, {
    error: "La nota puede tener como máximo un decimal",
  });

/** Inferred type from the grade value schema. */
export type GradeValue = z.infer<typeof gradeValueSchema>;
