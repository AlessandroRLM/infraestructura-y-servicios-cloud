import { z } from "zod";

/**
 * URL search schema for the admin section-enrollments view.
 * q: free-text search for the section selection table.
 * pageSize: must be 20 | 50 | 100; anything else → 50.
 */
export const sectionEnrollmentsSearchSchema = z.object({
  q: z.string().default("").catch(""),
  pageSize: z.coerce
    .number()
    .pipe(z.union([z.literal(20), z.literal(50), z.literal(100)]))
    .default(50)
    .catch(50),
});

export type SectionEnrollmentsSearch = z.infer<
  typeof sectionEnrollmentsSearchSchema
>;

/**
 * URL search schema for the student own section-enrollments view.
 * No filters — students see all their section enrollments unconditionally.
 * pageSize: must be 20 | 50 | 100; anything else → 20.
 */
export const ownSectionEnrollmentsSearchSchema = z.object({
  pageSize: z.coerce
    .number()
    .pipe(z.union([z.literal(20), z.literal(50), z.literal(100)]))
    .default(20)
    .catch(20),
});

export type OwnSectionEnrollmentsSearch = z.infer<
  typeof ownSectionEnrollmentsSearchSchema
>;
