import { useMutation } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import { CatalogService } from "@/gen/catalog/v1/catalog_pb";
import { PROGRAM_COURSES_QUERY_KEY } from "../api/queries";

export function useAddCourseToProgram(programId: string) {
  const queryClient = useQueryClient();
  return useMutation(CatalogService.method.addCourseToProgram, {
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: PROGRAM_COURSES_QUERY_KEY(programId),
      }),
  });
}
