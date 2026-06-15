/** Query-key constants for own-grades queries. Used by hooks and tests. */

/** Base key for all own-grades queries; used to invalidate the whole feature. */
export const OWN_GRADES_QUERY_KEY_BASE = ["grades", "own"] as const;

/**
 * Returns the infinite-query key for `listOwnGrades` scoped to the given
 * period and program filters. Changing either filter produces a distinct cache
 * entry and resets pagination to page 1.
 */
export function ownGradesQueryKey(
  academicPeriodId: string,
  programId: string,
): readonly [string, string, string, string] {
  return ["grades", "own", academicPeriodId, programId] as const;
}

/** Query key for `listOwnGradePeriods`. */
export const OWN_GRADE_PERIODS_QUERY_KEY = ["grades", "own-periods"] as const;

/** Query key for `listOwnEnrollments` used as carrera filter options. */
export const OWN_ENROLLMENTS_FOR_FILTER_QUERY_KEY = [
  "grades",
  "enrollments-filter",
] as const;
