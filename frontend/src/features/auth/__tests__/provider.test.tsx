import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it } from "vitest";
import { SessionProvider } from "../context/provider";
import { hasPermission, hasRole, useSession } from "../hooks/useSession";
import type { AuthenticatedSession, SessionSource } from "../types";

function makeQueryClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0, staleTime: 0 } },
  });
}

function makeSource(data: AuthenticatedSession | null): SessionSource {
  return { getSession: async () => data };
}

const pendingSource: SessionSource = {
  // Never settles — keeps the bootstrap query in pending state.
  getSession: () => new Promise(() => undefined),
};

function SessionConsumer() {
  const session = useSession();
  return (
    <div>
      <span data-testid="status">{session.status}</span>
      <span data-testid="user-id">
        {session.status === "authenticated" ? session.userId : "none"}
      </span>
    </div>
  );
}

function renderWithProvider(source: SessionSource, ui: ReactNode) {
  return render(
    <QueryClientProvider client={makeQueryClient()}>
      <SessionProvider source={source}>{ui}</SessionProvider>
    </QueryClientProvider>,
  );
}

describe("SessionProvider", () => {
  it("provides the authenticated session to children via useSession()", async () => {
    const source = makeSource({
      userId: "u-1",
      email: "test@example.com",
      roles: ["teacher"],
      permissions: ["grades.write"],
    });

    renderWithProvider(source, <SessionConsumer />);

    await waitFor(() =>
      expect(screen.getByTestId("status")).toHaveTextContent(/^authenticated$/),
    );
    expect(screen.getByTestId("user-id")).toHaveTextContent("u-1");
  });

  it("exposes loading while the bootstrap query is pending", () => {
    renderWithProvider(pendingSource, <SessionConsumer />);

    expect(screen.getByTestId("status")).toHaveTextContent("loading");
  });

  it("exposes unauthenticated when the source resolves to null", async () => {
    renderWithProvider(makeSource(null), <SessionConsumer />);

    await waitFor(() =>
      expect(screen.getByTestId("status")).toHaveTextContent(
        /^unauthenticated$/,
      ),
    );
  });

  it("exposes error when the source rejects", async () => {
    const errorSource: SessionSource = {
      getSession: () => Promise.reject(new Error("network failure")),
    };

    renderWithProvider(errorSource, <SessionConsumer />);

    await waitFor(() =>
      expect(screen.getByTestId("status")).toHaveTextContent(/^error$/),
    );
  });

  it("passes hasPermission and hasRole checks through the provider path", async () => {
    const source = makeSource({
      userId: "u-2",
      email: "admin@example.com",
      roles: ["admin"],
      permissions: ["catalog.manage"],
    });

    function GateConsumer() {
      const session = useSession();
      return (
        <div>
          <span data-testid="perm">
            {hasPermission(session, "catalog.manage") ? "yes" : "no"}
          </span>
          <span data-testid="role">
            {hasRole(session, "admin") ? "yes" : "no"}
          </span>
        </div>
      );
    }

    renderWithProvider(source, <GateConsumer />);

    await waitFor(() =>
      expect(screen.getByTestId("perm")).toHaveTextContent("yes"),
    );
    expect(screen.getByTestId("role")).toHaveTextContent("yes");
  });
});

describe("useSession() outside SessionProvider", () => {
  it("throws an explicit error when called outside the provider", () => {
    function BareConsumer() {
      useSession();
      return null;
    }

    expect(() => render(<BareConsumer />)).toThrow(
      "useSession must be used within a SessionProvider",
    );
  });
});
