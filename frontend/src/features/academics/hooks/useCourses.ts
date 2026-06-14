import { useTransport } from "@connectrpc/connect-query";
import { useQuery } from "@tanstack/react-query";
import { coursesQueryOptions } from "../api/queries";
import { createRpcCoursesSource } from "../api/rpc";

/**
 * Returns the flat Course[] from ListCourses.
 * The data projection lives here — the seam for swapping to useCursorList
 * once the backend adds pagination params to ListCoursesRequest.
 */
export function useCourses() {
  const transport = useTransport();
  const source = createRpcCoursesSource(transport);
  const result = useQuery(coursesQueryOptions(source));
  const courses = result.data ?? [];
  return {
    courses,
    isLoading: result.isLoading,
    isError: result.isError,
    refetch: result.refetch,
  };
}
