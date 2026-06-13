import { createRouter } from "@tanstack/react-router";
import { transport } from "./core/connect/transport";
import { queryClient } from "./core/query/queryClient";
import { createRpcSessionSource } from "./features/auth";
import { routeTree } from "./routeTree.gen";

// Single app-wide source instance: SessionProvider and route guards must hit
// the same query cache entry.
export const sessionSource = createRpcSessionSource(transport);

export const router = createRouter({
  routeTree,
  context: {
    queryClient,
    sessionSource,
  },
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
