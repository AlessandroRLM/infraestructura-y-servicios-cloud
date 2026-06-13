import { z } from "zod";

export const loginSchema = z.object({
  email: z.email({ error: "Correo electrónico no válido" }),
  password: z.string().min(1, { error: "La contraseña es obligatoria" }),
});

export type LoginValues = z.infer<typeof loginSchema>;
