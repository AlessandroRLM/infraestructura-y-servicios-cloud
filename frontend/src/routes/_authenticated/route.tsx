import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";
import { bootstrapQueryOptions, stubSessionSource } from "@/core/session";

export const Route = createFileRoute("/_authenticated")({
  beforeLoad: ({ context }) => {
    // Guard: treat null/undefined (no session injected) and explicitly
    // unauthenticated sessions as equivalent — redirect to /login.
    // Live session is injected per render via RouterProvider's context prop
    // (auth-and-guards Intent skill pattern).
    // The stub always returns unauthenticated, so _authenticated routes are
    // unreachable until the real SessionSource (GetSession RPC) lands.
    if (!context.session || context.session.status !== "authenticated") {
      throw redirect({ to: "/login" });
    }
  },
  loader: ({ context }) =>
    context.queryClient.ensureQueryData(
      bootstrapQueryOptions(stubSessionSource),
    ),
  component: () => <Outlet />,
});
