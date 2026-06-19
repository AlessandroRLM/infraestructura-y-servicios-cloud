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

// Extends the history state shape so navigate() accepts a `section` payload.
// GradesPage passes the full TeachingSection on row click so GradesSectionPage
// can render instantly without re-fetching ListOwnSections on click-through.
declare module "@tanstack/history" {
  interface HistoryState {
    section?: Record<string, unknown>;
  }
}
