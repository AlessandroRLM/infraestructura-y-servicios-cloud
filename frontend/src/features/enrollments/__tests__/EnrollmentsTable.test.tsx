/**
 * EnrollmentsTable component tests (admin view).
 *
 * ADR-5: Table reads Route.useSearch() + Route.useNavigate() — tests use
 * renderWithProviders at /admin/enrollments so the real route provides context.
 *
 * Covers:
 *  - Loading skeleton (aria-busy).
 *  - Populated rows + columns including studentName / programName from wire.
 *  - UUID-prefix fallback when studentName / programName are empty.
 *  - Empty state + Crear CTA for managers.
 *  - Initial transport error + Reintentar (no raw codes) + retry refetches.
 *  - Cargar más appends rows; prior rows remain.
 *  - No Cargar más when nextPageToken empty.
 *  - Fetch-next failure shows toast.
 *  - Deep-link ?q=foo → first listEnrollments call has query="foo".
 *  - Typing in search is debounced (≤2 new calls for 5 keystrokes).
 *  - Clearing search → query="".
 *  - Status __all__ → status="" in request.
 *  - Year filter reaches the request.
 *  - Status-aware action columns: pending → both; paid → Cancelar only; cancelled → none.
 */
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
  EnrollmentSchema,
  EnrollmentService,
  ListEnrollmentsResponseSchema,
} from "@/gen/enrollment/v1/enrollment_pb";
import { renderWithProviders } from "@/test";

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

// --- fixtures ---

const adminSession = {
  status: "authenticated" as const,
  userId: "admin-1",
  email: "admin@test.com",
  roles: ["admin"],
  permissions: ["enrollment.manage", "users.manage"],
};

const adminSessionSource = {
  getSession: async (): Promise<AuthenticatedSession> => ({
    userId: adminSession.userId,
    email: adminSession.email,
    roles: adminSession.roles,
    permissions: adminSession.permissions,
  }),
};

const enrollmentPending = create(EnrollmentSchema, {
  id: "e-pending",
  studentId: "aaaaaaaa-0000-0000-0000-000000000000",
  programId: "bbbbbbbb-0000-0000-0000-000000000000",
  year: 2026,
  status: "pending",
  createdAt: "2026-01-10T00:00:00Z",
  updatedAt: "2026-01-10T00:00:00Z",
  studentName: "Ana García",
  programName: "Ingeniería Civil",
});

const enrollmentPaid = create(EnrollmentSchema, {
  id: "e-paid",
  studentId: "cccccccc-0000-0000-0000-000000000000",
  programId: "dddddddd-0000-0000-0000-000000000000",
  year: 2025,
  status: "paid",
  paidAt: "2025-03-01T00:00:00Z",
  createdAt: "2025-01-10T00:00:00Z",
  updatedAt: "2025-03-01T00:00:00Z",
  studentName: "Bob López",
  programName: "Ingeniería Industrial",
});

const enrollmentCancelled = create(EnrollmentSchema, {
  id: "e-cancelled",
  studentId: "eeeeeeee-0000-0000-0000-000000000000",
  programId: "ffffffff-0000-0000-0000-000000000000",
  year: 2024,
  status: "cancelled",
  createdAt: "2024-01-10T00:00:00Z",
  updatedAt: "2024-06-01T00:00:00Z",
  studentName: "Carol Pérez",
  programName: "Derecho",
});

// Enrollment with empty wire names — should show UUID prefix fallback
const enrollmentNoNames = create(EnrollmentSchema, {
  id: "e-nonames",
  studentId: "12345678-aaaa-bbbb-cccc-000000000000",
  programId: "87654321-dddd-eeee-ffff-000000000000",
  year: 2026,
  status: "pending",
  createdAt: "2026-01-10T00:00:00Z",
  updatedAt: "2026-01-10T00:00:00Z",
  studentName: "",
  programName: "",
});

type EnrollmentImpl = Partial<ServiceImpl<typeof EnrollmentService>>;

function renderPage(
  handlers: EnrollmentImpl,
  permissions: string[] = ["enrollment.manage", "users.manage"],
) {
  return renderWithProviders({
    route: "/admin/enrollments",
    transport: makeStubTransport([EnrollmentService, handlers]),
    session: { ...adminSession, permissions },
    sessionSource: adminSessionSource,
  });
}

function renderPageWithRoute(route: string, handlers: EnrollmentImpl) {
  return renderWithProviders({
    route,
    transport: makeStubTransport([EnrollmentService, handlers]),
    session: adminSession,
    sessionSource: adminSessionSource,
  });
}

// --- tests ---

describe("EnrollmentsTable — loading", () => {
  it("shows aria-busy skeleton while listEnrollments is pending", async () => {
    // biome-ignore lint/suspicious/noEmptyBlockStatements: intentional never-resolving promise for loading state test
    renderPage({ listEnrollments: () => new Promise(() => {}) });

    const skeleton = await screen.findByRole("status", {
      name: "Cargando matrículas",
    });
    expect(skeleton).toBeInTheDocument();
    expect(skeleton).toHaveAttribute("aria-busy", "true");
  });
});

describe("EnrollmentsTable — populated rows", () => {
  it("shows studentName and programName from wire in correct columns", async () => {
    renderPage({
      listEnrollments: async () =>
        create(ListEnrollmentsResponseSchema, {
          enrollments: [enrollmentPending],
          nextPageToken: "",
        }),
    });

    await screen.findByText("Ana García");
    expect(screen.getByText("Ingeniería Civil")).toBeInTheDocument();
    expect(screen.getByText("2026")).toBeInTheDocument();
    // Status badge
    expect(screen.getByText("Pendiente")).toBeInTheDocument();
    // Pagado column: no paidAt → "—"
    expect(screen.getByText("—")).toBeInTheDocument();
  });

  it("shows UUID prefix fallback when studentName / programName are empty", async () => {
    renderPage({
      listEnrollments: async () =>
        create(ListEnrollmentsResponseSchema, {
          enrollments: [enrollmentNoNames],
          nextPageToken: "",
        }),
    });

    // studentId[:8] = "12345678", programId[:8] = "87654321"
    await screen.findByText("12345678");
    expect(screen.getByText("87654321")).toBeInTheDocument();
  });

  it("paid enrollment shows paidAt date and no Marcar pagada action", async () => {
    renderPage({
      listEnrollments: async () =>
        create(ListEnrollmentsResponseSchema, {
          enrollments: [enrollmentPaid],
          nextPageToken: "",
        }),
    });

    await screen.findByText("Bob López");
    // paidAt is set → shows a date (not "—")
    expect(screen.queryByText("—")).not.toBeInTheDocument();
    // "Marcar pagada" button must NOT be rendered for a paid enrollment
    expect(
      screen.queryByRole("button", { name: /marcar pagada/i }),
    ).not.toBeInTheDocument();
    // "Cancelar" button IS rendered for a paid enrollment
    expect(
      screen.getByRole("button", { name: /cancelar/i }),
    ).toBeInTheDocument();
  });
});

describe("EnrollmentsTable — empty state", () => {
  it("shows empty copy and Crear CTA when manager", async () => {
    renderPage({
      listEnrollments: async () =>
        create(ListEnrollmentsResponseSchema, {
          enrollments: [],
          nextPageToken: "",
        }),
    });

    await screen.findByText("Todavía no hay matrículas");
    const createButtons = screen.getAllByRole("button", {
      name: /crear matrícula/i,
    });
    expect(createButtons.length).toBeGreaterThanOrEqual(1);
  });
});

describe("EnrollmentsTable — error state", () => {
  it("shows error message and Reintentar; no raw error code", async () => {
    renderPage({
      listEnrollments: async () => {
        throw new ConnectError("unavailable", Code.Unavailable);
      },
    });

    await screen.findByText("No se pudo cargar la lista de matrículas.");
    expect(
      screen.getByRole("button", { name: /reintentar/i }),
    ).toBeInTheDocument();

    expect(screen.queryByText(/Unavailable/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Code\./)).not.toBeInTheDocument();
  });

  it("clicking Reintentar refetches and renders rows on success", async () => {
    const user = userEvent.setup();
    let callCount = 0;
    const listEnrollments = vi.fn(async () => {
      callCount++;
      if (callCount === 1) {
        throw new ConnectError("unavailable", Code.Unavailable);
      }
      return create(ListEnrollmentsResponseSchema, {
        enrollments: [enrollmentPending],
        nextPageToken: "",
      });
    });

    renderPage({ listEnrollments });

    await screen.findByRole("button", { name: /reintentar/i });
    await user.click(screen.getByRole("button", { name: /reintentar/i }));

    await screen.findByText("Ana García");
    expect(listEnrollments).toHaveBeenCalledTimes(2);
  });
});

describe("EnrollmentsTable — pagination", () => {
  it("Cargar más appends enrollments; prior rows remain", async () => {
    const user = userEvent.setup();
    let callCount = 0;
    const listEnrollments = vi.fn(async () => {
      callCount++;
      if (callCount === 1) {
        return create(ListEnrollmentsResponseSchema, {
          enrollments: [enrollmentPending],
          nextPageToken: "cursor-page-2",
        });
      }
      return create(ListEnrollmentsResponseSchema, {
        enrollments: [enrollmentPaid],
        nextPageToken: "",
      });
    });

    renderPage({ listEnrollments });

    await screen.findByText("Ana García");
    expect(
      screen.getByRole("button", { name: /cargar más/i }),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /cargar más/i }));

    await screen.findByText("Bob López");
    expect(screen.getByText("Ana García")).toBeInTheDocument();
  });

  it("no Cargar más when nextPageToken is empty", async () => {
    renderPage({
      listEnrollments: async () =>
        create(ListEnrollmentsResponseSchema, {
          enrollments: [enrollmentPending],
          nextPageToken: "",
        }),
    });

    await screen.findByText("Ana García");
    expect(
      screen.queryByRole("button", { name: /cargar más/i }),
    ).not.toBeInTheDocument();
  });

  it("fetchNextPage failure shows toast; existing rows remain", async () => {
    let callCount = 0;
    renderPage({
      listEnrollments: vi.fn(async () => {
        callCount++;
        if (callCount === 1) {
          return create(ListEnrollmentsResponseSchema, {
            enrollments: [enrollmentPending],
            nextPageToken: "cursor-page-2",
          });
        }
        throw new ConnectError("unavailable", Code.Unavailable);
      }),
    });

    const user = userEvent.setup();
    await screen.findByText("Ana García");

    await user.click(screen.getByRole("button", { name: /cargar más/i }));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith(
        "No se pudieron cargar más matrículas.",
      );
    });
    // Existing rows remain
    expect(screen.getByText("Ana García")).toBeInTheDocument();
    expect(
      screen.queryByText("No se pudo cargar la lista de matrículas."),
    ).not.toBeInTheDocument();
  });
});

describe("EnrollmentsTable — search and filters", () => {
  it("deep-link ?q=foo → first listEnrollments call has query='foo'", async () => {
    const listEnrollments = vi.fn(async () =>
      create(ListEnrollmentsResponseSchema, {
        enrollments: [],
        nextPageToken: "",
      }),
    );

    renderPageWithRoute("/admin/enrollments?q=foo", { listEnrollments });

    await waitFor(() => {
      expect(listEnrollments).toHaveBeenCalled();
    });

    const calls = listEnrollments.mock.calls as unknown as Array<
      [{ query: string }]
    >;
    expect(calls[0][0].query).toBe("foo");
    const emptyQueryCalls = calls.filter((c) => c[0].query === "");
    expect(emptyQueryCalls).toHaveLength(0);
  });

  it("typing in search is debounced (≤2 new calls for 5 keystrokes)", async () => {
    const user = userEvent.setup();
    const listEnrollments = vi.fn(async () =>
      create(ListEnrollmentsResponseSchema, {
        enrollments: [],
        nextPageToken: "",
      }),
    );

    renderPage({ listEnrollments });

    await screen.findByText("Todavía no hay matrículas");
    const initialCallCount = listEnrollments.mock.calls.length;

    const input = screen.getByPlaceholderText(
      /buscar por estudiante o programa/i,
    );
    await user.type(input, "alice");

    await waitFor(
      () => {
        const allCalls = listEnrollments.mock.calls as unknown as Array<
          [{ query: string }]
        >;
        const lastCall = allCalls[allCalls.length - 1];
        expect(lastCall?.[0].query).toBe("alice");
      },
      { timeout: 1000 },
    );

    const newCallCount = listEnrollments.mock.calls.length - initialCallCount;
    expect(newCallCount).toBeLessThanOrEqual(2);
  });

  it("clearing search resets query to ''", async () => {
    const user = userEvent.setup();
    const listEnrollments = vi.fn(async () =>
      create(ListEnrollmentsResponseSchema, {
        enrollments: [],
        nextPageToken: "",
      }),
    );

    renderPageWithRoute("/admin/enrollments?q=alice", { listEnrollments });

    await screen.findByText("Todavía no hay matrículas");

    const input = screen.getByPlaceholderText(
      /buscar por estudiante o programa/i,
    );
    await user.clear(input);

    await waitFor(
      () => {
        const allCalls = listEnrollments.mock.calls as unknown as Array<
          [{ query: string }]
        >;
        const lastCall = allCalls[allCalls.length - 1];
        expect(lastCall?.[0].query).toBe("");
      },
      { timeout: 1000 },
    );
  });

  it("year filter reaches listEnrollments request", async () => {
    const listEnrollments = vi.fn(async () =>
      create(ListEnrollmentsResponseSchema, {
        enrollments: [],
        nextPageToken: "",
      }),
    );

    renderPageWithRoute("/admin/enrollments?year=2025", { listEnrollments });

    await waitFor(() => {
      expect(listEnrollments).toHaveBeenCalled();
    });

    const calls = listEnrollments.mock.calls as unknown as Array<
      [{ year: number }]
    >;
    expect(calls[0][0].year).toBe(2025);
  });

  it("status __all__ (no URL status) → status='' in request", async () => {
    const listEnrollments = vi.fn(async () =>
      create(ListEnrollmentsResponseSchema, {
        enrollments: [],
        nextPageToken: "",
      }),
    );

    // No ?status param → adminEnrollmentsSearchSchema defaults status to undefined
    renderPage({ listEnrollments });

    await waitFor(() => {
      expect(listEnrollments).toHaveBeenCalled();
    });

    const calls = listEnrollments.mock.calls as unknown as Array<
      [{ status: string }]
    >;
    expect(calls[0][0].status).toBe("");
  });
});

describe("EnrollmentsTable — status-aware actions", () => {
  it("pending row shows Marcar pagada + Cancelar", async () => {
    renderPage({
      listEnrollments: async () =>
        create(ListEnrollmentsResponseSchema, {
          enrollments: [enrollmentPending],
          nextPageToken: "",
        }),
    });

    await screen.findByText("Ana García");
    expect(
      screen.getByRole("button", { name: /marcar pagada/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /cancelar/i }),
    ).toBeInTheDocument();
  });

  it("paid row shows only Cancelar, not Marcar pagada", async () => {
    renderPage({
      listEnrollments: async () =>
        create(ListEnrollmentsResponseSchema, {
          enrollments: [enrollmentPaid],
          nextPageToken: "",
        }),
    });

    await screen.findByText("Bob López");
    expect(
      screen.queryByRole("button", { name: /marcar pagada/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /cancelar/i }),
    ).toBeInTheDocument();
  });

  it("cancelled row shows no action buttons", async () => {
    renderPage({
      listEnrollments: async () =>
        create(ListEnrollmentsResponseSchema, {
          enrollments: [enrollmentCancelled],
          nextPageToken: "",
        }),
    });

    await screen.findByText("Carol Pérez");
    expect(
      screen.queryByRole("button", { name: /marcar pagada/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /cancelar/i }),
    ).not.toBeInTheDocument();
  });
});
