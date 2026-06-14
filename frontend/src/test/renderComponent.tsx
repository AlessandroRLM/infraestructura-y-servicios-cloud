import type { Transport } from "@connectrpc/connect";
import { TransportProvider } from "@connectrpc/connect-query";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { type RenderResult, render } from "@testing-library/react";
import type { ReactNode } from "react";
import { transport as defaultTransport } from "@/core/connect/transport";
import { SessionContext, type SessionState } from "@/features/auth";

interface RenderComponentOptions {
  transport?: Transport;
  session?: SessionState;
}

/**
 * Renders a component in isolation with the data + session providers but NO
 * router. Use for unit-testing a component's own behavior (e.g. permission-
 * driven affordances) independent of route guards. For anything that exercises
 * routing, use `renderWithProviders` instead.
 */
export function renderComponent(
  ui: ReactNode,
  options: RenderComponentOptions = {},
): RenderResult {
  const {
    transport = defaultTransport,
    session = { status: "unauthenticated" },
  } = options;

  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0, staleTime: 0 } },
  });

  return render(
    <TransportProvider transport={transport}>
      <QueryClientProvider client={queryClient}>
        <SessionContext value={session}>{ui}</SessionContext>
      </QueryClientProvider>
    </TransportProvider>,
  );
}
