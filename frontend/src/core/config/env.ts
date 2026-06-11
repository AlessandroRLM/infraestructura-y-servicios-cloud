import { z } from "zod";

const schema = z.object({
  VITE_API_URL: z.url(),
});

export const env = schema.parse(import.meta.env);
