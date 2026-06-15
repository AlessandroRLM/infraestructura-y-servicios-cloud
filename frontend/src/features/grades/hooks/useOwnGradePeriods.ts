import { useTransport } from "@connectrpc/connect-query";
import { useQuery } from "@tanstack/react-query";
import type { GradePeriod } from "@/gen/grades/v1/grades_pb";
import { OWN_GRADE_PERIODS_QUERY_KEY } from "../api/queries";
import { createRpcOwnGradesSource } from "../api/rpc";
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
  const transport = useTransport();
  const source = createRpcOwnGradesSource(transport);

  const result = useQuery({
    queryKey: OWN_GRADE_PERIODS_QUERY_KEY,
    queryFn: () => source.listOwnGradePeriods(),
    staleTime: 60_000,
  });

  const periods: PeriodOption[] = (result.data ?? ([] as GradePeriod[])).map(
    (p) => ({
      id: p.academicPeriodId,
      label: formatPeriod(p.year, p.term),
    }),
  );

  return {
    periods,
    isLoading: result.isLoading,
  };
}
