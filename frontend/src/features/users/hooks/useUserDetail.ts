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
      isError: profileQuery.isError,
      refetch: profileQuery.refetch,
    },
    student: {
      data: studentQuery.data,
      isLoading: studentQuery.isLoading,
      isError: studentQuery.isError,
      refetch: studentQuery.refetch,
    },
    teacher: {
      data: teacherQuery.data,
      isLoading: teacherQuery.isLoading,
      isError: teacherQuery.isError,
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
