import { useMutation } from "@connectrpc/connect-query";
import { createConnectQueryKey } from "@connectrpc/connect-query-core";
import { useQueryClient } from "@tanstack/react-query";
import { EnrollmentService } from "@/gen/enrollment/v1/enrollment_pb";

/**
 * Cancels a pending or paid enrollment. On success, invalidates the entire
 * EnrollmentService query cache (covers listEnrollments + listOwnEnrollments).
 */
export function useCancelEnrollment() {
  const queryClient = useQueryClient();
  return useMutation(EnrollmentService.method.cancelEnrollment, {
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: createConnectQueryKey({
          schema: EnrollmentService,
          cardinality: undefined,
        }),
      }),
  });
}
