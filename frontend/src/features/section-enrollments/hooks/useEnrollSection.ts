import { useMutation } from "@connectrpc/connect-query";
import { createConnectQueryKey } from "@connectrpc/connect-query-core";
import { useQueryClient } from "@tanstack/react-query";
import { SectionEnrollmentService } from "@/gen/section_enrollment/v1/section_enrollment_pb";

/**
 * Admin mutation: enroll a student (by enrollment_id) into a section.
 * On success, invalidates the entire SectionEnrollmentService query cache.
 */
export function useEnrollSection() {
  const queryClient = useQueryClient();
  return useMutation(SectionEnrollmentService.method.enrollSection, {
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: createConnectQueryKey({
          schema: SectionEnrollmentService,
          cardinality: undefined,
        }),
      }),
  });
}
