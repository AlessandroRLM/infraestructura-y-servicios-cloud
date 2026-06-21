import { z } from "zod";

/**
 * Form schema for creating or editing a program quota.
 * programId is NOT a user-editable field — it is injected by the caller
 * (always the program already in context). Only year and admissionQuota are
 * filled by the user.
 */
export const programQuotaSchema = z.object({
  year: z
    .number({ error: "Ingresa el año" })
    .int({ error: "El año debe ser un número entero" })
    .min(2000, { error: "El año mínimo es 2000" })
    .max(2100, { error: "El año máximo es 2100" }),
  admissionQuota: z
    .number({ error: "Ingresa el cupo" })
    .int({ error: "El cupo debe ser un número entero" })
    .min(1, { error: "El cupo mínimo es 1" })
    .max(10000, { error: "El cupo máximo es 10000" }),
});

export type ProgramQuotaFormValues = z.infer<typeof programQuotaSchema>;
