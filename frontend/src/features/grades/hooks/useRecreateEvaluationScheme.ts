import { useMutation } from "@connectrpc/connect-query";
import { createConnectQueryKey } from "@connectrpc/connect-query-core";
import { useQueryClient } from "@tanstack/react-query";
import { GradesService } from "@/gen/grades/v1/grades_pb";

/**
 * Mutation hook for RecreateEvaluationScheme.
 * On success, invalidates the ListEvaluations query for the given courseId
 * so the scheme display re-fetches and shows the replacement evaluations.
 *
 * Errors are passed through — the caller (SchemeManagementView) is responsible
 * for catching FailedPrecondition (locked scheme) and surfacing it via
 * mapSchemeError. This hook never swallows errors.
 *
 * Key form: same targeted-method key as useCreateEvaluationScheme.
 *
 * @param courseId - UUID of the course whose scheme is being replaced.
 */
export function useRecreateEvaluationScheme(courseId: string) {
  const queryClient = useQueryClient();
  return useMutation(GradesService.method.recreateEvaluationScheme, {
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: createConnectQueryKey({
          schema: GradesService.method.listEvaluations,
          input: { courseId },
          cardinality: "finite",
        }),
      }),
  });
}
