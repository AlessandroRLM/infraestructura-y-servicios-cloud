import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";
import { bootstrapQueryOptions } from "@/features/auth";

export const Route = createFileRoute("/_authenticated")({
  // Awaiting the bootstrap query (instead of reading a session snapshot)
  // makes navigations during the loading window wait for the real answer
  // rather than redirecting to /login prematurely. Same query key and
  // staleTime as SessionProvider — one cache entry, no second fetch.
  beforeLoad: async ({ context }) => {
    const session = await context.queryClient.ensureQueryData(
      bootstrapQueryOptions(context.sessionSource),
    );
    if (!session) {
      throw redirect({ to: "/login" });
    }
  },
  component: () => <Outlet />,
});
