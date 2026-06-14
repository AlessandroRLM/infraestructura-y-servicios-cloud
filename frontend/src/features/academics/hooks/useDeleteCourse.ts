import { useMutation } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import { CatalogService } from "@/gen/catalog/v1/catalog_pb";
import { COURSES_QUERY_KEY } from "../api/queries";

export function useDeleteCourse() {
  const queryClient = useQueryClient();
  return useMutation(CatalogService.method.deleteCourse, {
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: COURSES_QUERY_KEY }),
  });
}
