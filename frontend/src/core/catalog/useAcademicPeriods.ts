import { useQuery } from "@connectrpc/connect-query";
import { CatalogService } from "@/gen/catalog/v1/catalog_pb";

/**
 * Fetches the full list of academic periods (unary — no pagination in the RPC).
 * Returns a stable sorted list: most recent year+term first.
 */
export function useAcademicPeriods() {
  const result = useQuery(CatalogService.method.listAcademicPeriods, {});

  const periods = (result.data?.academicPeriods ?? []).slice().sort((a, b) => {
    if (b.year !== a.year) return b.year - a.year;
    return b.term - a.term;
  });

  return {
    periods,
    isLoading: result.isLoading,
    isError: result.isError,
    refetch: result.refetch,
  };
}

/**
 * Returns a human-readable label for a period, e.g. "2026 · Semestre 1".
 */
export function academicPeriodLabel(year: number, term: number): string {
  return `${year} · Semestre ${term}`;
}
