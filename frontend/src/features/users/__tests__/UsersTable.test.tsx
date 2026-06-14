import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import type { AuthenticatedSession } from "@/features/auth";
import {
  IamService,
  ListUsersResponseSchema,
  UserStatus,
  UserSummarySchema,
} from "@/gen/iam/v1/iam_pb";
import { renderWithProviders } from "@/test";

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

const adminSession = {
  status: "authenticated" as const,
  userId: "u-admin",
  email: "admin@test.com",
  roles: ["admin"],
  permissions: ["users.manage"],
};

// Prevents the _authenticated guard from redirecting to /login when navigation
// (search param update) triggers ensureQueryData with staleTime:0 in test QueryClient.
const adminSessionSource = {
  getSession: async (): Promise<AuthenticatedSession> => ({
    userId: adminSession.userId,
    email: adminSession.email,
    roles: adminSession.roles,
    permissions: adminSession.permissions,
  }),
};

const user1 = create(UserSummarySchema, {
  id: "u1",
  email: "alice@test.com",
  displayName: "Alice Smith",
  roles: ["student"],
  status: UserStatus.ACTIVE,
});

const user2 = create(UserSummarySchema, {
  id: "u2",
  email: "bob@test.com",
  displayName: "",
  roles: ["teacher"],
  status: UserStatus.DISABLED,
});

const user3 = create(UserSummarySchema, {
  id: "u3",
  email: "carol@test.com",
  displayName: "Carol",
  roles: [],
  status: UserStatus.UNSPECIFIED,
});

type IamImpl = Partial<ServiceImpl<typeof IamService>>;

function renderPage(
  handlers: IamImpl,
  permissions: string[] = ["users.manage"],
) {
  return renderWithProviders({
    route: "/users",
    transport: makeStubTransport([IamService, handlers]),
    session: { ...adminSession, permissions },
    sessionSource: adminSessionSource,
  });
}

function renderPageWithRoute(route: string, handlers: IamImpl) {
  return renderWithProviders({
    route,
    transport: makeStubTransport([IamService, handlers]),
    session: adminSession,
    sessionSource: adminSessionSource,
  });
}

describe("UsersTable", () => {
  it("S-01: shows aria-busy skeleton while listUsers is pending", async () => {
    // biome-ignore lint/suspicious/noEmptyBlockStatements: intentional never-resolving promise for loading state test
    renderPage({ listUsers: () => new Promise(() => {}) });

    const skeleton = await screen.findByRole("status", {
      name: "Cargando usuarios",
    });
    expect(skeleton).toBeInTheDocument();
    expect(skeleton).toHaveAttribute("aria-busy", "true");
  });

  it("S-01: rows appear after listUsers resolves", async () => {
    renderPage({
      listUsers: async () =>
        create(ListUsersResponseSchema, {
          users: [user1],
          nextPageToken: "",
        }),
    });

    await screen.findByText("alice@test.com");
    expect(screen.getByText("Alice Smith")).toBeInTheDocument();
  });

  it("S-02: shows empty state when listUsers returns []", async () => {
    renderPage({
      listUsers: async () =>
        create(ListUsersResponseSchema, { users: [], nextPageToken: "" }),
    });

    await screen.findByText("No se encontraron usuarios.");
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
  });

  it("S-03: shows error state + retry when listUsers fails, retry calls again", async () => {
    const user = userEvent.setup();
    let callCount = 0;
    const listUsers = vi.fn(async () => {
      callCount++;
      if (callCount === 1) {
        throw new ConnectError("unavailable", Code.Unavailable);
      }
      return create(ListUsersResponseSchema, {
        users: [user1],
        nextPageToken: "",
      });
    });

    renderPage({ listUsers });

    await screen.findByText("No se pudo cargar la lista de usuarios.");
    expect(
      screen.getByRole("button", { name: /reintentar/i }),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /reintentar/i }));
    await screen.findByText("alice@test.com");
    expect(listUsers).toHaveBeenCalledTimes(2);
  });

  it("S-04: debounce — listUsers called with typed query after debounce", async () => {
    const user = userEvent.setup();
    const listUsers = vi.fn(async () =>
      create(ListUsersResponseSchema, { users: [], nextPageToken: "" }),
    );

    renderPage({ listUsers });

    await screen.findByText("No se encontraron usuarios.");

    const input = screen.getByPlaceholderText(/buscar/i);
    await user.type(input, "alice");

    await waitFor(
      () => {
        const allCalls = listUsers.mock.calls as unknown as Array<
          [{ query: string }]
        >;
        const lastCall = allCalls[allCalls.length - 1];
        expect(lastCall?.[0].query).toBe("alice");
      },
      { timeout: 1000 },
    );
  });

  it("S-05: deep-link /users?q=bob — first listUsers call has query bob", async () => {
    const listUsers = vi.fn(async () =>
      create(ListUsersResponseSchema, { users: [], nextPageToken: "" }),
    );

    renderPageWithRoute("/users?q=bob", { listUsers });

    await waitFor(() => {
      expect(listUsers).toHaveBeenCalled();
    });

    const calls = listUsers.mock.calls as unknown as Array<[{ query: string }]>;
    expect(calls[0][0].query).toBe("bob");
    const emptyQueryCalls = calls.filter((c) => c[0].query === "");
    expect(emptyQueryCalls).toHaveLength(0);
  });

  it("S-06: clearing input resets list to empty query", async () => {
    const user = userEvent.setup();
    const listUsers = vi.fn(async () =>
      create(ListUsersResponseSchema, { users: [], nextPageToken: "" }),
    );

    renderPageWithRoute("/users?q=alice", { listUsers });

    await screen.findByText("No se encontraron usuarios.");

    const input = screen.getByPlaceholderText(/buscar/i);
    await user.clear(input);

    await waitFor(
      () => {
        const allCalls = listUsers.mock.calls as unknown as Array<
          [{ query: string }]
        >;
        const lastCall = allCalls[allCalls.length - 1];
        expect(lastCall?.[0].query).toBe("");
      },
      { timeout: 1000 },
    );
  });

  it("S-07: Cargar más appends users, prior rows remain", async () => {
    const user = userEvent.setup();
    let callCount = 0;
    const listUsers = vi.fn(async () => {
      callCount++;
      if (callCount === 1) {
        return create(ListUsersResponseSchema, {
          users: [user1],
          nextPageToken: "cursor-page-2",
        });
      }
      return create(ListUsersResponseSchema, {
        users: [user2],
        nextPageToken: "",
      });
    });

    renderPage({ listUsers });

    await screen.findByText("alice@test.com");
    expect(
      screen.getByRole("button", { name: /cargar más/i }),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /cargar más/i }));

    await screen.findAllByText("bob@test.com");
    expect(screen.getByText("alice@test.com")).toBeInTheDocument();
  });

  it("S-08: no Cargar más when no nextPageToken", async () => {
    renderPage({
      listUsers: async () =>
        create(ListUsersResponseSchema, {
          users: [user1],
          nextPageToken: "",
        }),
    });

    await screen.findByText("alice@test.com");
    expect(
      screen.queryByRole("button", { name: /cargar más/i }),
    ).not.toBeInTheDocument();
  });

  it("S-10: ACTIVE status shows Activo badge", async () => {
    renderPage({
      listUsers: async () =>
        create(ListUsersResponseSchema, {
          users: [user1],
          nextPageToken: "",
        }),
    });

    await screen.findByText("Activo");
  });

  it("S-11: DISABLED status shows Deshabilitado badge", async () => {
    renderPage({
      listUsers: async () =>
        create(ListUsersResponseSchema, {
          users: [user2],
          nextPageToken: "",
        }),
    });

    await screen.findAllByText("bob@test.com");
    expect(screen.getByText("Deshabilitado")).toBeInTheDocument();
  });

  it("S-12: UNSPECIFIED status renders as Activo", async () => {
    renderPage({
      listUsers: async () =>
        create(ListUsersResponseSchema, {
          users: [user3],
          nextPageToken: "",
        }),
    });

    await screen.findByText("carol@test.com");
    expect(screen.getByText("Activo")).toBeInTheDocument();
  });

  it("S-13: row click opens Sheet dialog", async () => {
    const user = userEvent.setup();
    renderPage({
      listUsers: async () =>
        create(ListUsersResponseSchema, {
          users: [user1],
          nextPageToken: "",
        }),
      getUser: async () => ({ user: user1 }),
    });

    await screen.findByText("alice@test.com");
    await user.click(screen.getByText("alice@test.com").closest("tr")!);

    await waitFor(() => {
      expect(screen.getByRole("dialog")).toBeInTheDocument();
    });
  });

  it("S-13: displayName falls back to email when displayName is empty", async () => {
    renderPage({
      listUsers: async () =>
        create(ListUsersResponseSchema, {
          users: [user2],
          nextPageToken: "",
        }),
    });

    await screen.findAllByText("bob@test.com");
    const cells = screen.getAllByText("bob@test.com");
    expect(cells.length).toBeGreaterThanOrEqual(2);
  });

  it("S-20: session without users.manage — table not rendered, listUsers not called", async () => {
    const listUsers = vi.fn(async () =>
      create(ListUsersResponseSchema, { users: [user1], nextPageToken: "" }),
    );

    renderPage({ listUsers }, []);

    await waitFor(() => {
      expect(screen.queryByPlaceholderText(/buscar/i)).not.toBeInTheDocument();
    });
    expect(listUsers).not.toHaveBeenCalled();
  });

  it("C-01: fetchNextPage failure shows toast, inline list-load error is separate", async () => {
    let callCount = 0;
    renderPage({
      listUsers: vi.fn(async () => {
        callCount++;
        if (callCount === 1) {
          return create(ListUsersResponseSchema, {
            users: [user1],
            nextPageToken: "cursor-page-2",
          });
        }
        throw new ConnectError("unavailable", Code.Unavailable);
      }),
    });

    const user = userEvent.setup();
    await screen.findByText("alice@test.com");

    await user.click(screen.getByRole("button", { name: /cargar más/i }));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith(
        "No se pudieron cargar más usuarios.",
      );
    });
    // Existing rows remain visible — no full-page error state
    expect(screen.getByText("alice@test.com")).toBeInTheDocument();
    expect(
      screen.queryByText("No se pudo cargar la lista de usuarios."),
    ).not.toBeInTheDocument();
  });

  it("W-01: getUser does NOT fire before a row is selected (Sheet closed, no userId)", async () => {
    const getUser = vi.fn(async () => ({ user: user1 }));

    renderPage({
      listUsers: async () =>
        create(ListUsersResponseSchema, {
          users: [user1],
          nextPageToken: "",
        }),
      getUser,
    });

    await screen.findByText("alice@test.com");

    // Sheet is closed (userId is undefined) — IAM detail query must not have fired
    expect(getUser).not.toHaveBeenCalled();
  });

  it("W-02: search change after page 2 resets to page 1 rows only", async () => {
    const user = userEvent.setup();
    let callCount = 0;
    renderPage({
      listUsers: vi.fn(async () => {
        callCount++;
        // First call: page 1 of default query
        if (callCount === 1) {
          return create(ListUsersResponseSchema, {
            users: [user1],
            nextPageToken: "cursor-page-2",
          });
        }
        // Second call: page 2 (fetchNextPage)
        if (callCount === 2) {
          return create(ListUsersResponseSchema, {
            users: [user2],
            nextPageToken: "",
          });
        }
        // Subsequent calls: new search query — page 1 only
        return create(ListUsersResponseSchema, {
          users: [user3],
          nextPageToken: "",
        });
      }),
    });

    // Wait for page 1
    await screen.findByText("alice@test.com");

    // Load page 2
    await user.click(screen.getByRole("button", { name: /cargar más/i }));
    await screen.findAllByText("bob@test.com");
    expect(screen.getByText("alice@test.com")).toBeInTheDocument();

    // Change search → should reset to page 1 of new query
    const input = screen.getByPlaceholderText(/buscar/i);
    await user.clear(input);
    await user.type(input, "carol");

    // Page 2 rows (bob) should be gone; new page 1 rows (carol) should appear
    await screen.findByText("carol@test.com");
    expect(screen.queryByText("bob@test.com")).not.toBeInTheDocument();
    expect(screen.queryByText("alice@test.com")).not.toBeInTheDocument();
  });

  it("W-03: debounce — intermediate keystrokes do not each fire listUsers", async () => {
    const user = userEvent.setup();
    const listUsers = vi.fn(async () =>
      create(ListUsersResponseSchema, { users: [], nextPageToken: "" }),
    );

    renderPage({ listUsers });

    // Wait for initial load
    await screen.findByText("No se encontraron usuarios.");
    const initialCallCount = listUsers.mock.calls.length;

    const input = screen.getByPlaceholderText(/buscar/i);
    await user.type(input, "alice");

    await waitFor(
      () => {
        const allCalls = listUsers.mock.calls as unknown as Array<
          [{ query: string }]
        >;
        const lastCall = allCalls[allCalls.length - 1];
        expect(lastCall?.[0].query).toBe("alice");
      },
      { timeout: 1000 },
    );

    // The final query "alice" should have fired, but NOT once per keystroke.
    // "alice" has 5 chars → if each fired we'd have initialCallCount + 5.
    // With debounce, total new calls must be well below that (≤ 2 is a safe bound).
    const newCallCount = listUsers.mock.calls.length - initialCallCount;
    expect(newCallCount).toBeLessThanOrEqual(2);
  });

  it("SG-01: row click opens Sheet without changing the URL", async () => {
    const user = userEvent.setup();
    const { router } = renderPage({
      listUsers: async () =>
        create(ListUsersResponseSchema, {
          users: [user1],
          nextPageToken: "",
        }),
      getUser: async () => ({ user: user1 }),
    });

    await screen.findByText("alice@test.com");
    const locationBefore = router.state.location.href;

    await user.click(screen.getByText("alice@test.com").closest("tr")!);

    await waitFor(() => {
      expect(screen.getByRole("dialog")).toBeInTheDocument();
    });

    // URL must not have changed (no $userId param, no ?user= query)
    expect(router.state.location.href).toBe(locationBefore);
  });
});
