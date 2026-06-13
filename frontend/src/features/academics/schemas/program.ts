import { z } from "zod";

export const programSchema = z.object({
  code: z.string().min(1, { error: "El código es obligatorio" }),
  name: z.string().min(1, { error: "El nombre es obligatorio" }),
});

export type ProgramFormValues = z.infer<typeof programSchema>;
