import type { Transport } from "@connectrpc/connect";
import { TransportProvider } from "@connectrpc/connect-query";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  createMemoryHistory,
  createRouter,
  RouterProvider,
} from "@tanstack/react-router";
import { type RenderResult, render } from "@testing-library/react";
import type { ReactNode } from "react";
import { transport as defaultTransport } from "@/core/connect/transport";
import { SessionCtx, type SessionData } from "@/core/session";
import { routeTree } from "../../routeTree.gen";

/** Unauthenticated fallback for tests that do not inject a session. */
const UNAUTHENTICATED_SESSION: SessionData = {
  user: null,
  roles: [],
  permissions: [],
  status: "unauthenticated",
};

interface RenderOptions {
  transport?: Transport;
  route?: string;
  /**
   * Inject a session for the test.
   *
   * The value is provided via SessionCtx.Provider (so useSession() works in
   * tested components) AND injected into the router context (so route guards
   * work). Both paths receive the same object — no duplication.
   *
   * Use an authenticated session to reach protected routes; omit or pass
   * unauthenticated to test redirect behavior.
   */
  session?: SessionData;
}

export function renderWithProviders(
  _ui: ReactNode,
  options: RenderOptions = {},
): RenderResult {
  const {
    transport = defaultTransport,
    route = "/",
    session = UNAUTHENTICATED_SESSION,
  } = options;

  const testQueryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0, staleTime: 0 },
    },
  });

  const memoryHistory = createMemoryHistory({ initialEntries: [route] });

  const router = createRouter({
    routeTree,
    history: memoryHistory,
    context: {
      queryClient: testQueryClient,
      session: undefined!,
    },
  });

  return render(
    <TransportProvider transport={transport}>
      <QueryClientProvider client={testQueryClient}>
        {/* SessionCtx.Provider makes useSession() work in tested components */}
        <SessionCtx.Provider value={session}>
          <RouterProvider router={router} context={{ session }} />
        </SessionCtx.Provider>
      </QueryClientProvider>
    </TransportProvider>,
  );
}
