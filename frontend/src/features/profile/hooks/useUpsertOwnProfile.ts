import { useMutation } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import { ProfileService } from "@/gen/profiles/v1/profiles_pb";
import { OWN_PROFILE_QUERY_KEY } from "../api/queries";

export function useUpsertOwnProfile() {
  const queryClient = useQueryClient();
  return useMutation(ProfileService.method.upsertOwnProfile, {
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: OWN_PROFILE_QUERY_KEY }),
  });
}
