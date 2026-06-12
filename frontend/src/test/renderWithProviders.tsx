import type { Transport } from "@connectrpc/connect";
import { TransportProvider } from "@connectrpc/connect-query";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  createMemoryHistory,
  createRouter,
  RouterProvider,
} from "@tanstack/react-router";
import { type RenderResult, render } from "@testing-library/react";
import { transport as defaultTransport } from "@/core/connect/transport";
import {
  type AuthenticatedSession,
  SESSION_QUERY_KEY,
  SessionContext,
  type SessionState,
  stubSessionSource,
} from "@/features/auth";
import { routeTree } from "../routeTree.gen";

interface RenderOptions {
  transport?: Transport;
  route?: string;
  /**
   * Session for the test: provided through SessionContext (so useSession()
   * works) and pre-seeded into the query cache (so route guards resolve
   * without fetching). With `loading` the cache is left unseeded, so the
   * _authenticated guard fetches from stubSessionSource, gets null, and
   * redirects to /login. With `error` the cache is seeded with null, so
   * guards behave as unauthenticated (redirect to /login); the error state
   * is only visible to useSession consumers rendered outside guarded routes.
   */
  session?: SessionState;
}

export function renderWithProviders(options: RenderOptions = {}): RenderResult {
  const {
    transport = defaultTransport,
    route = "/",
    session = { status: "unauthenticated" },
  } = options;

  const testQueryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0, staleTime: 0 },
    },
  });

  if (session.status !== "loading") {
    const seed: AuthenticatedSession | null =
      session.status === "authenticated"
        ? {
            userId: session.userId,
            email: session.email,
            roles: session.roles,
            permissions: session.permissions,
          }
        : null;
    testQueryClient.setQueryData(SESSION_QUERY_KEY, seed);
  }

  const memoryHistory = createMemoryHistory({ initialEntries: [route] });

  const router = createRouter({
    routeTree,
    history: memoryHistory,
    context: {
      queryClient: testQueryClient,
      sessionSource: stubSessionSource,
    },
  });

  return render(
    <TransportProvider transport={transport}>
      <QueryClientProvider client={testQueryClient}>
        <SessionContext value={session}>
          <RouterProvider router={router} />
        </SessionContext>
      </QueryClientProvider>
    </TransportProvider>,
  );
}
