import { queryOptions } from "@tanstack/react-query";
import type { UsersDetailSource } from "./rpc";

export const USER_DETAIL_BASE_KEY = ["users", "detail"] as const;

export function getUserQueryOptions(source: UsersDetailSource, userId: string) {
  return queryOptions({
    queryKey: [...USER_DETAIL_BASE_KEY, userId, "iam"] as const,
    queryFn: () => source.getUser(userId),
    staleTime: 30_000,
  });
}

export function getUserProfileQueryOptions(
  source: UsersDetailSource,
  userId: string,
) {
  return queryOptions({
    queryKey: [...USER_DETAIL_BASE_KEY, userId, "profile"] as const,
    queryFn: () => source.getUserProfile(userId),
    staleTime: 30_000,
  });
}

export function getStudentProfileQueryOptions(
  source: UsersDetailSource,
  userId: string,
) {
  return queryOptions({
    queryKey: [...USER_DETAIL_BASE_KEY, userId, "student"] as const,
    queryFn: () => source.getStudentProfile(userId),
    staleTime: 30_000,
  });
}

export function getTeacherProfileQueryOptions(
  source: UsersDetailSource,
  userId: string,
) {
  return queryOptions({
    queryKey: [...USER_DETAIL_BASE_KEY, userId, "teacher"] as const,
    queryFn: () => source.getTeacherProfile(userId),
    staleTime: 30_000,
  });
}

export function listTeacherQualificationsQueryOptions(
  source: UsersDetailSource,
  userId: string,
) {
  return queryOptions({
    queryKey: [...USER_DETAIL_BASE_KEY, userId, "qualifications"] as const,
    queryFn: () => source.listTeacherQualifications(userId),
    staleTime: 30_000,
  });
}
