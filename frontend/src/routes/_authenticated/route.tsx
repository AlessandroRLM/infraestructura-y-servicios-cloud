import { createFileRoute, redirect } from "@tanstack/react-router";
import { AuthenticatedLayout } from "@/components/layout/AuthenticatedLayout";
import { bootstrapQueryOptions, eligibleAreas } from "@/features/auth";

export const Route = createFileRoute("/_authenticated")({
  // Awaiting the bootstrap query (instead of reading a session snapshot)
  // makes navigations during the loading window wait for the real answer
  // rather than redirecting to /login prematurely. Same query key and
  // staleTime as SessionProvider — one cache entry, no second fetch.
  beforeLoad: async ({ context, location }) => {
    const session = await context.queryClient.ensureQueryData(
      bootstrapQueryOptions(context.sessionSource),
    );
    if (!session) {
      throw redirect({ to: "/login", search: { redirect: location.href } });
    }
    // Resolve both the session and its area eligibility once here so all
    // child routes read from context instead of recomputing.
    return { session, eligibility: eligibleAreas(session) };
  },
  component: AuthenticatedLayout,
});
