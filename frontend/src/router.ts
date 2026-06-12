import { createRouter } from "@tanstack/react-router";
import { queryClient } from "./core/query/queryClient";
import { routeTree } from "./routeTree.gen";

// Router is created once with a placeholder session.
// Live session state is injected per render via RouterProvider's context prop
// in the app root component (see main.tsx).
// auth-and-guards Intent skill: never recreate the router on auth changes.
export const router = createRouter({
  routeTree,
  context: {
    queryClient,
    session: undefined!,
  },
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
