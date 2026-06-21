import { useInfiniteQuery } from "@connectrpc/connect-query";
import { DEFAULT_PAGE_SIZE } from "@/core/pagination";
import { EnrollmentService } from "@/gen/enrollment/v1/enrollment_pb";

/**
 * Fetches paid enrollments for the admin section-enrollment picker.
 * Backed by EnrollmentService.listEnrollments with status="paid".
 * Used exclusively by EnrollSectionDialog to allow the admin to pick
 * which annual enrollment to link to the section.
 */
export function useEnrollmentsForPicker(pageSize = DEFAULT_PAGE_SIZE) {
  const result = useInfiniteQuery(
    EnrollmentService.method.listEnrollments,
    {
      year: 0,
      status: "paid",
      studentId: "",
      programId: "",
      pageSize,
      pageToken: "",
    },
    {
      pageParamKey: "pageToken",
      getNextPageParam: (lastPage) => lastPage.nextPageToken || undefined,
    },
  );

  const enrollments =
    result.data?.pages.flatMap((page) => page.enrollments) ?? [];

  return {
    enrollments,
    isLoading: result.isLoading,
  };
}
