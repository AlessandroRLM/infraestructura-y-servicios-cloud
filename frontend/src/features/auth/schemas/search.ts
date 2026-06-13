import { z } from "zod";

// redirect carries the destination the guard bounced the user from; default
// to the app root when login is reached directly.
export const loginSearchSchema = z.object({
  // Allowlist: accept only clean internal paths; reject anything that is not.
  redirect: z
    .string()
    // biome-ignore lint/suspicious/noControlCharactersInRegex: intentional — rejects 0x00–0x1F control chars as redirect bypass vectors
    .transform((val) => (/^\/(?![/\\])[^\s\x00-\x1F]*$/.test(val) ? val : "/"))
    .default("/"),
});
