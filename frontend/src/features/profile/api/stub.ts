import { create } from "@bufbuild/protobuf";
import { UserProfileSchema } from "@/gen/profiles/v1/profiles_pb";
import type { ProfileSource } from "./rpc";

export function stubProfileSource(
  overrides?: Partial<ProfileSource>,
): ProfileSource {
  return {
    getOwnProfile: async () => create(UserProfileSchema, {}),
    ...overrides,
  };
}
