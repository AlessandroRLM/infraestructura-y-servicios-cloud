import { z } from "zod";

/**
 * URL search schema for the admin enrollments view.
 * q: free-text search (student name/email, program code/name).
 * year: optional 4-digit year (2000–2100); out-of-range → undefined.
 * status: optional enrollment status; invalid value → undefined.
 * pageSize: must be 20 | 50 | 100; anything else → 20.
 * studentId / programId are intentionally omitted (search supersedes them).
 */
export const adminEnrollmentsSearchSchema = z.object({
  q: z.string().default("").catch(""),
  year: z.coerce.number().int().min(2000).max(2100).optional().catch(undefined),
  status: z.enum(["pending", "paid", "cancelled"]).optional().catch(undefined),
  pageSize: z.coerce
    .number()
    .pipe(z.union([z.literal(20), z.literal(50), z.literal(100)]))
    .catch(20),
});

export type AdminEnrollmentsSearch = z.infer<
  typeof adminEnrollmentsSearchSchema
>;

/**
 * URL search schema for the student own-enrollments view.
 * No filters — students see all their enrollments unconditionally.
 * pageSize: must be 20 | 50 | 100; anything else → 20.
 */
export const ownEnrollmentsSearchSchema = z.object({
  pageSize: z.coerce
    .number()
    .pipe(z.union([z.literal(20), z.literal(50), z.literal(100)]))
    .catch(20),
});

export type OwnEnrollmentsSearch = z.infer<typeof ownEnrollmentsSearchSchema>;
