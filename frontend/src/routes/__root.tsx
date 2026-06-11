import type { QueryClient } from "@tanstack/react-query";
import { createRootRouteWithContext, Outlet } from "@tanstack/react-router";
import type { SessionContext } from "@/core/session/context";

interface RouterContext {
  queryClient: QueryClient;
  session: SessionContext | null;
}

export const Route = createRootRouteWithContext<RouterContext>()({
  component: () => <Outlet />,
});
