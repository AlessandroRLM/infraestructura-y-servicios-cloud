import { useMutation } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import { AuthService } from "@/gen/auth/v1/auth_pb";
import { SESSION_QUERY_KEY } from "../api/queries";

export function useLogin() {
  const queryClient = useQueryClient();
  return useMutation(AuthService.method.login, {
    // Await the invalidation so mutateAsync resolves only once the session is
    // marked stale; the post-login navigation then re-runs the guard against
    // fresh data instead of the pre-login null.
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: SESSION_QUERY_KEY }),
  });
}
