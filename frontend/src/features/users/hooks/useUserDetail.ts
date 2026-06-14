import { Code, ConnectError } from "@connectrpc/connect";
import { useQuery } from "@tanstack/react-query";
import {
  getStudentProfileQueryOptions,
  getTeacherProfileQueryOptions,
  getUserProfileQueryOptions,
  getUserQueryOptions,
  listTeacherQualificationsQueryOptions,
} from "../api/queries";
import type { UsersDetailSource } from "../api/rpc";

// A missing profile/teacher/student row is a normal state (the user simply
// hasn't been enrolled yet), not a failure — surface it apart from real errors.
function isNotFoundError(query: { isError: boolean; error: unknown }): boolean {
  return (
    query.isError &&
    query.error instanceof ConnectError &&
    query.error.code === Code.NotFound
  );
}

export function useUserDetail(userId: string, source: UsersDetailSource) {
  const iamQuery = useQuery({
    ...getUserQueryOptions(source, userId),
    enabled: !!userId,
  });

  const isNotFound =
    (iamQuery.isSuccess && !iamQuery.data?.user) ||
    (iamQuery.isError &&
      iamQuery.error instanceof ConnectError &&
      iamQuery.error.code === Code.NotFound);

  const roles = iamQuery.data?.user?.roles ?? [];
  const hasStudentRole = !isNotFound && roles.includes("student");
  const hasTeacherRole = !isNotFound && roles.includes("teacher");

  const profileQuery = useQuery({
    ...getUserProfileQueryOptions(source, userId),
    enabled: !!userId,
  });

  const studentQuery = useQuery({
    ...getStudentProfileQueryOptions(source, userId),
    enabled: hasStudentRole,
  });

  const teacherQuery = useQuery({
    ...getTeacherProfileQueryOptions(source, userId),
    enabled: hasTeacherRole,
  });

  const qualsQuery = useQuery({
    ...listTeacherQualificationsQueryOptions(source, userId),
    enabled: hasTeacherRole,
  });

  const profileNotFound = isNotFoundError(profileQuery);
  const studentNotFound = isNotFoundError(studentQuery);
  const teacherNotFound = isNotFoundError(teacherQuery);

  return {
    iam: {
      data: iamQuery.data,
      isLoading: iamQuery.isLoading,
      isError: iamQuery.isError && !isNotFound,
      refetch: iamQuery.refetch,
    },
    profile: {
      data: profileQuery.data,
      isLoading: profileQuery.isLoading,
      isError: profileQuery.isError && !profileNotFound,
      isNotFound: profileNotFound,
      refetch: profileQuery.refetch,
    },
    student: {
      data: studentQuery.data,
      isLoading: studentQuery.isLoading,
      isError: studentQuery.isError && !studentNotFound,
      isNotFound: studentNotFound,
      refetch: studentQuery.refetch,
    },
    teacher: {
      data: teacherQuery.data,
      isLoading: teacherQuery.isLoading,
      isError: teacherQuery.isError && !teacherNotFound,
      isNotFound: teacherNotFound,
      refetch: teacherQuery.refetch,
    },
    quals: {
      data: qualsQuery.data,
      isLoading: qualsQuery.isLoading,
      isError: qualsQuery.isError,
      refetch: qualsQuery.refetch,
    },
    isNotFound,
  };
}
