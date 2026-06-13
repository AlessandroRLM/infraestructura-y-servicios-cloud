import { Code, ConnectError } from "@connectrpc/connect";
import { useMutation } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import { AuthService } from "@/gen/auth/v1/auth_pb";
import { SESSION_QUERY_KEY } from "../api/queries";

export function useLogout() {
  const queryClient = useQueryClient();
  return useMutation(AuthService.method.logout, {
    // The cookie is dead server-side; write null directly instead of refetching
    // a session that is guaranteed to come back unauthenticated.
    onSuccess: () => {
      queryClient.setQueryData(SESSION_QUERY_KEY, null);
    },
    // Unauthenticated means the cookie was already expired server-side;
    // clear the stale cache entry so the guard doesn't re-render an auth state.
    onError: (err) => {
      if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
        queryClient.setQueryData(SESSION_QUERY_KEY, null);
      }
    },
  });
}
