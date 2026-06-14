import { createClient, type Transport } from "@connectrpc/connect";
import type { UserProfile } from "@/gen/profiles/v1/profiles_pb";
import { ProfileService } from "@/gen/profiles/v1/profiles_pb";

export interface ProfileSource {
  getOwnProfile(): Promise<UserProfile>;
}

export function createRpcProfileSource(transport: Transport): ProfileSource {
  const client = createClient(ProfileService, transport);
  return {
    async getOwnProfile() {
      const res = await client.getOwnProfile({});
      return res;
    },
  };
}
