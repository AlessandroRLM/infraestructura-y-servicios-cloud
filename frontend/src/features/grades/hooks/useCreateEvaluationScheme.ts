import { useMutation } from "@connectrpc/connect-query";
import { createConnectQueryKey } from "@connectrpc/connect-query-core";
import { useQueryClient } from "@tanstack/react-query";
import { GradesService } from "@/gen/grades/v1/grades_pb";

/**
 * Mutation hook for CreateEvaluationScheme.
 * On success, invalidates the ListEvaluations query for the given courseId
 * so the scheme display re-fetches and shows the newly created evaluations.
 *
 * Key form used: targeted method key
 * `{ schema: GradesService.method.listEvaluations, input: { courseId }, cardinality: "finite" }`
 * This is supported by connect-query-core v2 (DescMethodUnary overload of
 * createConnectQueryKey). It targets only the listEvaluations(courseId) entry
 * and avoids invalidating unrelated grades queries.
 *
 * @param courseId - UUID of the course for which the scheme is being created.
 */
export function useCreateEvaluationScheme(courseId: string) {
  const queryClient = useQueryClient();
  return useMutation(GradesService.method.createEvaluationScheme, {
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
