import { useTransport } from "@connectrpc/connect-query";
import { useQuery } from "@tanstack/react-query";
import { programsQueryOptions } from "../api/queries";
import { createRpcProgramsSource } from "../api/rpc";

/**
 * Returns the flat Program[] from ListPrograms.
 * The data projection lives here — the seam for swapping to useCursorList
 * once the backend adds pagination params to ListProgramsRequest.
 */
export function usePrograms() {
  const transport = useTransport();
  const source = createRpcProgramsSource(transport);
  const result = useQuery(programsQueryOptions(source));
  const programs = result.data ?? [];
  return {
    programs,
    isLoading: result.isLoading,
    isError: result.isError,
    refetch: result.refetch,
  };
}
