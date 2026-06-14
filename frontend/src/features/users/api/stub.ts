// List stub: provide via makeStubTransport([IamService, { listUsers }]).
// This file stubs the 5 detail RPCs only (source-injected pattern).
import { create } from "@bufbuild/protobuf";
import { GetUserResponseSchema } from "@/gen/iam/v1/iam_pb";
import {
  StudentProfileSchema,
  TeacherProfileSchema,
  UserProfileSchema,
} from "@/gen/profiles/v1/profiles_pb";
import type { UsersDetailSource } from "./rpc";

export function stubUsersDetailSource(
  overrides?: Partial<UsersDetailSource>,
): UsersDetailSource {
  return {
    getUser: async () => create(GetUserResponseSchema, {}),
    getUserProfile: async () => create(UserProfileSchema, {}),
    getStudentProfile: async () => create(StudentProfileSchema, {}),
    getTeacherProfile: async () => create(TeacherProfileSchema, {}),
    listTeacherQualifications: async () => [],
    ...overrides,
  };
}
