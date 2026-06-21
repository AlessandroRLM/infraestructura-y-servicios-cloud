import { useMutation } from "@connectrpc/connect-query";
import { createConnectQueryKey } from "@connectrpc/connect-query-core";
import { useQueryClient } from "@tanstack/react-query";
import { SectionEnrollmentService } from "@/gen/section_enrollment/v1/section_enrollment_pb";

/**
 * Admin mutation: withdraw a student from a section (in_progress → withdrawn).
 * On success, invalidates the entire SectionEnrollmentService query cache.
 */
export function useWithdrawSection() {
  const queryClient = useQueryClient();
  return useMutation(SectionEnrollmentService.method.withdrawSection, {
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: createConnectQueryKey({
          schema: SectionEnrollmentService,
          cardinality: undefined,
        }),
      }),
  });
}
