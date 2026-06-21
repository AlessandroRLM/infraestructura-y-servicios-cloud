import { useMutation } from "@connectrpc/connect-query";
import { createConnectQueryKey } from "@connectrpc/connect-query-core";
import { useQueryClient } from "@tanstack/react-query";
import { CatalogService } from "@/gen/catalog/v1/catalog_pb";

export function useRemoveTeacher() {
  const queryClient = useQueryClient();
  return useMutation(CatalogService.method.removeTeacherFromSection, {
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: createConnectQueryKey({
          schema: CatalogService,
          cardinality: undefined,
        }),
      }),
  });
}
