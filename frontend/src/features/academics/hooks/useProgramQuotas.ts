import { useQuery } from "@connectrpc/connect-query";
import { CatalogService } from "@/gen/catalog/v1/catalog_pb";

/**
 * Fetches quotas for a specific program via ListProgramQuotas.
 * The query is disabled (no request issued) when programId is an empty string.
 * ListProgramQuotas REQUIRES a valid programId — there is no "list all" RPC.
 *
 * @param programId - UUID of the program. Pass "" to keep the query idle.
 */
export function useProgramQuotas(programId: string) {
  const result = useQuery(
    CatalogService.method.listProgramQuotas,
    { programId },
    { enabled: programId !== "" },
  );

  return {
    programQuotas: result.data?.programQuotas ?? [],
    isLoading: result.isLoading,
    isError: result.isError,
    refetch: result.refetch,
  };
}
