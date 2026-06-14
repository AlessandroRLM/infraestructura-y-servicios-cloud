import { z } from "zod";

export const profileSchema = z.object({
  birthDate: z
    .string()
    .regex(/^\d{4}-\d{2}-\d{2}$/, { error: "Formato AAAA-MM-DD" })
    .optional()
    .or(z.literal("")),
  phone: z.string().optional(),
  personalEmail: z
    .union([z.literal(""), z.email({ error: "Email inválido" })])
    .optional(),
  addressStreet: z.string().optional(),
  commune: z.string().optional(),
  region: z.string().optional(),
  country: z.string().optional(),
  postalCode: z.string().optional(),
  emergencyContactName: z.string().optional(),
  emergencyContactPhone: z.string().optional(),
});

export type ProfileFormValues = z.infer<typeof profileSchema>;
