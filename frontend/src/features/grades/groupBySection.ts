import type { OwnGrade } from "@/gen/grades/v1/grades_pb";

/** Raw status values as sent by the backend. */
export type SectionStatusRaw =
  | "in_progress"
  | "passed"
  | "failed"
  | "withdrawn";

/** Status-to-Spanish label map. Keys are exhaustive over the backend enum. */
export const STATUS_LABELS: Record<SectionStatusRaw, string> = {
  in_progress: "En curso",
  passed: "Aprobado",
  failed: "Reprobado",
  withdrawn: "Retirado",
} as const;

/**
 * Returns the Spanish display label for a raw section status value.
 * Unknown values fall through to a safe default.
 */
export function formatStatus(raw: string): string {
  if (raw in STATUS_LABELS) {
    return STATUS_LABELS[raw as SectionStatusRaw];
  }
  return raw;
}

/** Formats an academic period as "{year}-{term}". */
export function formatPeriod(year: number, term: number): string {
  return `${year}-${term}`;
}

/**
 * One evaluation row inside a section group.
 * Evaluation name is intentionally absent — there is no name field in the data model.
 */
export interface EvaluationRow {
  /** The 1-based ordinal of this evaluation in the course scheme. */
  position: number;
  /** Decimal weight string, e.g. "0.300". */
  weight: string;
  /** The student's recorded grade value, e.g. "5.5". */
  value: string;
}

/**
 * Grouped view of all OwnGrade rows that share a single section enrollment.
 * Section-level fields (course, period, status, final grade) are taken from
 * the first row in the group — the invariant is that all rows for the same
 * sectionEnrollmentId carry identical section-level data.
 */
export interface GradeSectionGroup {
  sectionEnrollmentId: string;
  courseCode: string;
  courseName: string;
  /** Formatted as "{year}-{term}". */
  period: string;
  /** Empty string from the backend renders as "—" in the UI. */
  finalGrade: string;
  /** Spanish display label derived via {@link formatStatus}. */
  status: string;
  /** Evaluations sorted ascending by position. */
  evaluations: EvaluationRow[];
}

/**
 * Groups a flat array of OwnGrade rows into section enrollment groups.
 * Insertion order is preserved (grades arrive newest-first from the API;
 * sections appear in the order their first row is encountered).
 * Evaluations within each group are sorted ascending by position.
 */
export function groupBySection(grades: OwnGrade[]): GradeSectionGroup[] {
  const map = new Map<string, GradeSectionGroup>();

  for (const grade of grades) {
    const existing = map.get(grade.sectionEnrollmentId);
    const evalRow: EvaluationRow = {
      position: grade.evaluationPosition,
      weight: grade.evaluationWeight,
      value: grade.value,
    };

    if (existing) {
      existing.evaluations.push(evalRow);
    } else {
      map.set(grade.sectionEnrollmentId, {
        sectionEnrollmentId: grade.sectionEnrollmentId,
        courseCode: grade.courseCode,
        courseName: grade.courseName,
        period: formatPeriod(
          grade.academicPeriodYear,
          grade.academicPeriodTerm,
        ),
        finalGrade: grade.sectionFinalGrade,
        status: formatStatus(grade.sectionStatus),
        evaluations: [evalRow],
      });
    }
  }

  // Sort evaluations within each group by ascending position.
  for (const group of map.values()) {
    group.evaluations.sort((a, b) => a.position - b.position);
  }

  return Array.from(map.values());
}
