import { z } from "zod";

/**
 * Tab values for the reports dashboard.
 * Each value corresponds to a report type gated by permissions.
 */
const REPORT_TABS = [
  "section-grade",
  "occupancy",
  "program-summary",
  "student-record",
] as const;

export type ReportTab = (typeof REPORT_TABS)[number];

/**
 * Zod 4 schema for the reports route search params.
 * All fields use .catch() so invalid URL params never crash the page.
 * Used as validateSearch in the TanStack Router route file.
 */
const reportsSearchSchema = z.object({
  tab: z.enum(REPORT_TABS).default("section-grade").catch("section-grade"),
  sectionId: z.string().default("").catch(""),
  periodId: z.string().default("").catch(""),
  programId: z.string().default("").catch(""),
  studentId: z.string().default("").catch(""),
  year: z.coerce.number().int().min(2000).max(2100).optional().catch(undefined),
});

export type ReportsSearch = z.infer<typeof reportsSearchSchema>;

/**
 * Validates and coerces route search params for the reports route.
 * Returns safe defaults for any missing or invalid fields.
 * Non-object inputs (null, number, string) are treated as empty and fall back
 * to all defaults — they never throw.
 */
export function validateSearch(input: unknown): ReportsSearch {
  const obj = input !== null && typeof input === "object" ? input : {};
  return reportsSearchSchema.parse(obj);
}
