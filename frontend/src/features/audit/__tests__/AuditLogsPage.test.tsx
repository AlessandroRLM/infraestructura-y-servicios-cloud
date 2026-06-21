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
  AuditLogSchema,
  AuditLogsService,
  ListRecentAuditLogsResponseSchema,
} from "@/gen/audit_logs/v1/audit_logs_pb";
import {
  IamService,
  ListUsersResponseSchema,
  UserStatus,
  UserSummarySchema,
} from "@/gen/iam/v1/iam_pb";
import {
  DisplayNameSchema,
  ListDisplayNamesByIDsResponseSchema,
  ProfileService,
} from "@/gen/profiles/v1/profiles_pb";
import { renderWithProviders } from "@/test";

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

const adminSession = {
  status: "authenticated" as const,
  userId: "u-admin",
  email: "admin@test.com",
  roles: ["admin"],
  permissions: ["audit.read"],
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

const log1 = create(AuditLogSchema, {
  id: "log-1",
  actorId: "u-alice",
  action: "grade.update",
  entity: "grades",
  entityId: "grade-1",
  detail: '{"old":1,"new":2}',
  createdAt: "2024-03-15T14:30:00Z",
});

const log2 = create(AuditLogSchema, {
  id: "log-2",
  actorId: "",
  action: "section.create",
  entity: "sections",
  entityId: "sec-1",
  detail: "",
  createdAt: "2024-03-14T09:00:00Z",
});

const log3 = create(AuditLogSchema, {
  id: "log-3",
  actorId: "u-bob",
  action: "user.disable",
  entity: "users",
  entityId: "u-bob",
  detail: "",
  createdAt: "2024-03-13T11:00:00Z",
});

const aliceName = create(DisplayNameSchema, {
  userId: "u-alice",
  givenNames: "Alice",
  lastNamePaternal: "Smith",
});

const aliceUser = create(UserSummarySchema, {
  id: "u-alice",
  email: "alice@test.com",
  displayName: "Alice Smith",
  roles: ["teacher"],
  status: UserStatus.ACTIVE,
});

type AuditImpl = Partial<ServiceImpl<typeof AuditLogsService>>;
type ProfileImpl = Partial<ServiceImpl<typeof ProfileService>>;
type IamImpl = Partial<ServiceImpl<typeof IamService>>;

function renderPage(
  auditHandlers: AuditImpl,
  profileHandlers: ProfileImpl = {},
  iamHandlers: IamImpl = {},
  route = "/admin/audit",
) {
  return renderWithProviders({
    route,
    transport: makeStubTransport(
      [AuditLogsService, auditHandlers],
      [ProfileService, profileHandlers],
      [IamService, iamHandlers],
    ),
    session: adminSession,
    sessionSource: adminSessionSource,
  });
}

describe("AuditLogsPage", () => {
  it("A-01: shows aria-busy skeleton while listRecentAuditLogs is pending", async () => {
    // biome-ignore lint/suspicious/noEmptyBlockStatements: intentional never-resolving promise for loading state test
    renderPage({ listRecentAuditLogs: () => new Promise(() => {}) });

    const skeleton = await screen.findByRole("status", {
      name: "Cargando registros de auditoría",
    });
    expect(skeleton).toBeInTheDocument();
    expect(skeleton).toHaveAttribute("aria-busy", "true");
  });

  it("A-02: rows appear after listRecentAuditLogs resolves; actor name shows", async () => {
    renderPage(
      {
        listRecentAuditLogs: async () =>
          create(ListRecentAuditLogsResponseSchema, {
            logs: [log1],
            nextPageToken: "",
          }),
      },
      {
        listDisplayNamesByIDs: async () =>
          create(ListDisplayNamesByIDsResponseSchema, {
            names: [aliceName],
          }),
      },
    );

    // Wait for table to appear
    await screen.findByText("grade.update");
    // Actor name resolved from display names
    expect(screen.getByText("Alice Smith")).toBeInTheDocument();
    // Columns
    expect(screen.getByText("grades")).toBeInTheDocument();
  });

  it("A-03: empty actorId shows 'Sistema'", async () => {
    renderPage({
      listRecentAuditLogs: async () =>
        create(ListRecentAuditLogsResponseSchema, {
          logs: [log2],
          nextPageToken: "",
        }),
    });

    await screen.findByText("section.create");
    expect(screen.getByText("Sistema")).toBeInTheDocument();
  });

  it("A-04: shows empty state when listRecentAuditLogs returns []", async () => {
    renderPage({
      listRecentAuditLogs: async () =>
        create(ListRecentAuditLogsResponseSchema, {
          logs: [],
          nextPageToken: "",
        }),
    });

    await screen.findByText("No se encontraron registros de auditoría.");
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
  });

  it("A-05: shows error + Reintentar when listRecentAuditLogs fails; retry calls again", async () => {
    const user = userEvent.setup();
    let callCount = 0;
    const listRecentAuditLogs = vi.fn(async () => {
      callCount++;
      if (callCount === 1) {
        throw new ConnectError("unavailable", Code.Unavailable);
      }
      return create(ListRecentAuditLogsResponseSchema, {
        logs: [log1],
        nextPageToken: "",
      });
    });

    renderPage({ listRecentAuditLogs });

    await screen.findByText("No se pudo cargar el registro de auditoría.");
    expect(
      screen.getByRole("button", { name: /reintentar/i }),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /reintentar/i }));
    await screen.findByText("grade.update");
    expect(listRecentAuditLogs).toHaveBeenCalledTimes(2);
  });

  it("A-06: Cargar más appends logs, prior rows remain", async () => {
    const user = userEvent.setup();
    let callCount = 0;
    const listRecentAuditLogs = vi.fn(async () => {
      callCount++;
      if (callCount === 1) {
        return create(ListRecentAuditLogsResponseSchema, {
          logs: [log1],
          nextPageToken: "cursor-page-2",
        });
      }
      return create(ListRecentAuditLogsResponseSchema, {
        logs: [log3],
        nextPageToken: "",
      });
    });

    renderPage({ listRecentAuditLogs });

    await screen.findByText("grade.update");
    expect(
      screen.getByRole("button", { name: /cargar más/i }),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /cargar más/i }));

    await screen.findByText("user.disable");
    expect(screen.getByText("grade.update")).toBeInTheDocument();
  });

  it("A-07: no Cargar más when no nextPageToken", async () => {
    renderPage({
      listRecentAuditLogs: async () =>
        create(ListRecentAuditLogsResponseSchema, {
          logs: [log1],
          nextPageToken: "",
        }),
    });

    await screen.findByText("grade.update");
    expect(
      screen.queryByRole("button", { name: /cargar más/i }),
    ).not.toBeInTheDocument();
  });

  it("A-08: fetchNextPage failure shows toast, loaded rows remain", async () => {
    const user = userEvent.setup();
    let callCount = 0;
    renderPage({
      listRecentAuditLogs: vi.fn(async () => {
        callCount++;
        if (callCount === 1) {
          return create(ListRecentAuditLogsResponseSchema, {
            logs: [log1],
            nextPageToken: "cursor-page-2",
          });
        }
        throw new ConnectError("unavailable", Code.Unavailable);
      }),
    });

    await screen.findByText("grade.update");
    await user.click(screen.getByRole("button", { name: /cargar más/i }));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith(
        "No se pudieron cargar más registros.",
      );
    });
    expect(screen.getByText("grade.update")).toBeInTheDocument();
    expect(
      screen.queryByText("No se pudo cargar el registro de auditoría."),
    ).not.toBeInTheDocument();
  });

  it("A-09: from date filter reaches listRecentAuditLogs as RFC3339 start-of-day", async () => {
    const listRecentAuditLogs = vi.fn(async () =>
      create(ListRecentAuditLogsResponseSchema, {
        logs: [],
        nextPageToken: "",
      }),
    );

    renderPage({ listRecentAuditLogs }, {}, {}, "/admin/audit?from=2024-03-01");

    await screen.findByText("No se encontraron registros de auditoría.");

    await waitFor(() => {
      expect(listRecentAuditLogs).toHaveBeenCalled();
    });

    const calls = listRecentAuditLogs.mock.calls as unknown as Array<
      [{ createdFrom: string }]
    >;
    const callsWithFilter = calls.filter(
      (c) => c[0].createdFrom === "2024-03-01T00:00:00Z",
    );
    expect(callsWithFilter.length).toBeGreaterThan(0);
  });

  it("A-10: actor filter is passed as actorId to listRecentAuditLogs", async () => {
    const listRecentAuditLogs = vi.fn(async () =>
      create(ListRecentAuditLogsResponseSchema, {
        logs: [],
        nextPageToken: "",
      }),
    );

    renderPage(
      { listRecentAuditLogs },
      {},
      {
        listUsers: async () =>
          create(ListUsersResponseSchema, {
            users: [aliceUser],
            nextPageToken: "",
          }),
      },
      "/admin/audit?actorId=u-alice",
    );

    await screen.findByText("No se encontraron registros de auditoría.");

    await waitFor(() => {
      expect(listRecentAuditLogs).toHaveBeenCalled();
    });

    const calls = listRecentAuditLogs.mock.calls as unknown as Array<
      [{ actorId: string }]
    >;
    const callsWithActor = calls.filter((c) => c[0].actorId === "u-alice");
    expect(callsWithActor.length).toBeGreaterThan(0);
  });
});
