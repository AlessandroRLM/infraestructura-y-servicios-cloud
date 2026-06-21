import { useMutation } from "@connectrpc/connect-query";
import { createConnectQueryKey } from "@connectrpc/connect-query-core";
import { useQueryClient } from "@tanstack/react-query";
import { SectionEnrollmentService } from "@/gen/section_enrollment/v1/section_enrollment_pb";

/**
 * Student self-enroll mutation: creates a section inscription for the authenticated student.
 * On success, invalidates the entire SectionEnrollmentService query cache so both the
 * enrollable-sections list and the own-enrollments list refresh.
 */
export function useEnrollOwnSection() {
  const queryClient = useQueryClient();
  return useMutation(SectionEnrollmentService.method.enrollOwnSection, {
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: createConnectQueryKey({
          schema: SectionEnrollmentService,
          cardinality: undefined,
        }),
      }),
  });
}
