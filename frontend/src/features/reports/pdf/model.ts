/**
 * Normalized model consumed by ReportPdfDocument and by all per-report mappers.
 * Views and hooks depend only on this type — never on the PDF library internals.
 */
export interface ReportPdfModel {
  title: string;
  appliedFilter: string;
  generatedAt: string;
  truncatedTo?: number;
  columns: {
    key: string;
    label: string;
    /** Relative width as a percentage of the table (columns sum to 100). */
    width: number;
    /** Cell text alignment; defaults to "left". Numeric columns use "right". */
    align?: "left" | "right";
  }[];
  rows: string[][];
  footer: string;
}
