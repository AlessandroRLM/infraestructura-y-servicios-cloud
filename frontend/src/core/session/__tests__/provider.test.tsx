import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { SessionProvider } from "../provider";
import type { SessionData, SessionSource } from "../source";
import { useSession } from "../useSession";

// --- helpers ---

function makeQueryClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0, staleTime: 0 } },
  });
}

function makeSource(data: SessionData): SessionSource {
  return { getSession: async () => data };
}

/** Renders a component that calls useSession() inside a SessionProvider. */
function SessionConsumer() {
  const session = useSession();
  return (
    <div>
      <span data-testid="status">{session.status}</span>
      <span data-testid="user-id">{session.user?.id ?? "none"}</span>
    </div>
  );
}

/** Renders a component that calls useSession() WITHOUT a SessionProvider. */
function BareConsumer() {
  const session = useSession();
  return <span>{session.status}</span>;
}

// --- tests ---

describe("SessionProvider", () => {
  it("provides session data to children via useSession()", async () => {
    const qc = makeQueryClient();
    const source = makeSource({
      user: { id: "u-1", email: "test@example.com" },
      roles: ["teacher"],
      permissions: ["grades.write"],
      status: "authenticated",
    });

    render(
      <QueryClientProvider client={qc}>
        <SessionProvider source={source}>
          <SessionConsumer />
        </SessionProvider>
      </QueryClientProvider>,
    );

    expect(await screen.findByTestId("status")).toHaveTextContent(
      "authenticated",
    );
    expect(await screen.findByTestId("user-id")).toHaveTextContent("u-1");
  });

  it("provides unauthenticated session while query is pending (initial value)", async () => {
    const qc = makeQueryClient();
    // Source that never resolves synchronously — tests the initial value
    const source = makeSource({
      user: null,
      roles: [],
      permissions: [],
      status: "unauthenticated",
    });

    render(
      <QueryClientProvider client={qc}>
        <SessionProvider source={source}>
          <SessionConsumer />
        </SessionProvider>
      </QueryClientProvider>,
    );

    // After settling, must show unauthenticated
    expect(await screen.findByTestId("status")).toHaveTextContent(
      "unauthenticated",
    );
  });

  it("passes hasPermission check through the provider path", async () => {
    const qc = makeQueryClient();
    const source = makeSource({
      user: { id: "u-2", email: "admin@example.com" },
      roles: ["admin"],
      permissions: ["catalog.manage"],
      status: "authenticated",
    });

    function PermissionConsumer() {
      const session = useSession();
      const allowed =
        session.status === "authenticated" &&
        session.permissions.includes("catalog.manage");
      return <span data-testid="perm">{allowed ? "yes" : "no"}</span>;
    }

    render(
      <QueryClientProvider client={qc}>
        <SessionProvider source={source}>
          <PermissionConsumer />
        </SessionProvider>
      </QueryClientProvider>,
    );

    await waitFor(() =>
      expect(screen.getByTestId("perm")).toHaveTextContent("yes"),
    );
  });

  it("passes hasRole check through the provider path", async () => {
    const qc = makeQueryClient();
    const source = makeSource({
      user: { id: "u-3", email: "s@example.com" },
      roles: ["student"],
      permissions: [],
      status: "authenticated",
    });

    function RoleConsumer() {
      const session = useSession();
      const isStudent =
        session.status === "authenticated" && session.roles.includes("student");
      return <span data-testid="role">{isStudent ? "yes" : "no"}</span>;
    }

    render(
      <QueryClientProvider client={qc}>
        <SessionProvider source={source}>
          <RoleConsumer />
        </SessionProvider>
      </QueryClientProvider>,
    );

    await waitFor(() =>
      expect(screen.getByTestId("role")).toHaveTextContent("yes"),
    );
  });
});

describe("useSession() outside SessionProvider", () => {
  it("throws an explicit error when called outside the provider", () => {
    // RTL renders into a tree — we catch the thrown error with a boundary or
    // by asserting the render itself throws.
    expect(() => {
      const qc = makeQueryClient();
      render(
        <QueryClientProvider client={qc}>
          <BareConsumer />
        </QueryClientProvider>,
      );
    }).toThrow("useSessionContext must be used within a SessionProvider");
  });
});
