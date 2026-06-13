import { useMutation } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import { CatalogService } from "@/gen/catalog/v1/catalog_pb";
import { PROGRAMS_QUERY_KEY } from "../api/queries";

export function useDeleteProgram() {
  const queryClient = useQueryClient();
  return useMutation(CatalogService.method.deleteProgram, {
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: PROGRAMS_QUERY_KEY }),
  });
}
