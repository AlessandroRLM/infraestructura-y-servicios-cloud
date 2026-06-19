/**
 * Centralized date formatter for the "generatedAt" timestamp.
 * Used by both the on-screen display (ReportPdfPreview) and the PDF document
 * (ReportPdfDocument) so they always render the same human-readable string.
 *
 * @param isoString - ISO 8601 timestamp, e.g. "2026-06-17T20:15:00Z"
 * @returns Formatted string, e.g. "17 de junio de 2026, 20:15"
 */
export function formatGeneratedAt(isoString: string): string {
  return new Intl.DateTimeFormat("es", {
    dateStyle: "long",
    timeStyle: "short",
  }).format(new Date(isoString));
}
