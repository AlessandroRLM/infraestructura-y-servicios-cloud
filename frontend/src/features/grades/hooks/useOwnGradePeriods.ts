import { useQuery } from "@connectrpc/connect-query";
import { GradesService } from "@/gen/grades/v1/grades_pb";
import { formatPeriod } from "../groupBySection";

/** One option in the período dropdown. */
export interface PeriodOption {
  /** The UUID of the academic period used as the filter value. */
  id: string;
  /** Display label formatted as "{year}-{term}". */
  label: string;
}

/** Query result for período dropdown options. */
export interface UseOwnGradePeriodsResult {
  periods: PeriodOption[];
  isLoading: boolean;
}

/**
 * Fetches the distinct academic periods in which the authenticated student
 * has grades. Used to populate the período filter dropdown.
 */
export function useOwnGradePeriods(): UseOwnGradePeriodsResult {
  const result = useQuery(GradesService.method.listOwnGradePeriods, {});

  const periods: PeriodOption[] = (result.data?.periods ?? []).map((p) => ({
    id: p.academicPeriodId,
    label: formatPeriod(p.year, p.term),
  }));

  return {
    periods,
    isLoading: result.isLoading,
  };
}
