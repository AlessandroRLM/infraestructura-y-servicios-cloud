import { createClient, type Transport } from "@connectrpc/connect";
import type { GetUserResponse } from "@/gen/iam/v1/iam_pb";
import { IamService } from "@/gen/iam/v1/iam_pb";
import type {
  StudentProfile,
  TeacherProfile,
  TeacherQualification,
  UserProfile,
} from "@/gen/profiles/v1/profiles_pb";
import { ProfileService } from "@/gen/profiles/v1/profiles_pb";

export interface UsersDetailSource {
  getUser(userId: string): Promise<GetUserResponse>;
  getUserProfile(userId: string): Promise<UserProfile>;
  getStudentProfile(userId: string): Promise<StudentProfile>;
  getTeacherProfile(userId: string): Promise<TeacherProfile>;
  listTeacherQualifications(teacherId: string): Promise<TeacherQualification[]>;
}

export function createRpcUsersDetailSource(
  transport: Transport,
): UsersDetailSource {
  const iamClient = createClient(IamService, transport);
  const profileClient = createClient(ProfileService, transport);
  return {
    async getUser(userId) {
      return iamClient.getUser({ userId });
    },
    async getUserProfile(userId) {
      return profileClient.getUserProfile({ userId });
    },
    async getStudentProfile(userId) {
      return profileClient.getStudentProfile({ userId });
    },
    async getTeacherProfile(userId) {
      return profileClient.getTeacherProfile({ userId });
    },
    async listTeacherQualifications(teacherId) {
      const res = await profileClient.listTeacherQualifications({ teacherId });
      return res.qualifications;
    },
  };
}
