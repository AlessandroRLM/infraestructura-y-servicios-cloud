/**
 * OwnEnrollmentsList component tests (student own-view).
 *
 * The list renders at /app/enrollments with the ownEnrollmentsSearchSchema
 * (pageSize only — no filters per ADR-8).
 *
 * Covers:
 *  - Renders programName from wire + year + EnrollmentStatusBadge.
 *  - "Pagar" button visible for pending rows; NOT for paid/cancelled rows.
 *  - Clicking "Pagar" opens the PayOwnEnrollmentDialog.
 *  - Loading skeleton (role=status, aria-busy).
 *  - Empty state message.
 *  - Initial error state + Reintentar button.
 *  - Cargar más appends rows; prior rows remain.
 *  - No Cargar más when nextPageToken is empty.
 *  - Cargar más failure shows toast.
 *  - PageSizeSelector present.
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

type EnrollmentImpl = Partial<ServiceImpl<typeof EnrollmentService>>;

const studentSession = {
  status: "authenticated" as const,
  userId: "s1",
  email: "student@test.com",
  roles: ["student"],
  permissions: ["enrollment.view_own"],
};

const studentSessionSource = {
  getSession: async (): Promise<AuthenticatedSession> => ({
    userId: studentSession.userId,
    email: studentSession.email,
    roles: studentSession.roles,
    permissions: studentSession.permissions,
  }),
};

function makeEnrollment(
  overrides: Partial<Parameters<typeof create<typeof EnrollmentSchema>>[1]> = {},
) {
  return create(EnrollmentSchema, {
    id: "enr-1",
    studentId: "s1",
    programId: "prog-1",
    programName: "Ingeniería Civil",
    studentName: "Juan Pérez",
    year: 2026,
    status: "pending",
    createdAt: "2026-01-15T00:00:00Z",
    updatedAt: "2026-01-15T00:00:00Z",
    ...overrides,
  });
}

function renderOwnEnrollments(
  handlers: EnrollmentImpl,
  route = "/app/enrollments?pageSize=20",
) {
  return renderWithProviders({
    route,
    transport: makeStubTransport([EnrollmentService, handlers]),
    session: studentSession,
    sessionSource: studentSessionSource,
  });
}

describe("OwnEnrollmentsList — data display", () => {
  it("renders programName from wire in the table", async () => {
    renderOwnEnrollments({
      listOwnEnrollments: async () =>
        create(ListEnrollmentsResponseSchema, {
          enrollments: [makeEnrollment({ programName: "Ingeniería Civil" })],
          nextPageToken: "",
        }),
    });

    await screen.findByText("Ingeniería Civil");
  });

  it("renders year and status badge", async () => {
    renderOwnEnrollments({
      listOwnEnrollments: async () =>
        create(ListEnrollmentsResponseSchema, {
          enrollments: [makeEnrollment({ year: 2026, status: "paid" })],
          nextPageToken: "",
        }),
    });

    await screen.findByText("2026");
    // Status badge renders "Pagada" for paid
    await screen.findByText("Pagada");
  });

  it("renders em-dash for paidAt when absent", async () => {
    renderOwnEnrollments({
      listOwnEnrollments: async () =>
        create(ListEnrollmentsResponseSchema, {
          enrollments: [makeEnrollment({ status: "pending", paidAt: undefined })],
          nextPageToken: "",
        }),
    });

    await screen.findByText("Ingeniería Civil");
    // paidAt column shows em-dash when absent
    expect(screen.getByText("—")).toBeInTheDocument();
  });

  it("shows a Pagar button for pending rows", async () => {
    renderOwnEnrollments({
      listOwnEnrollments: async () =>
        create(ListEnrollmentsResponseSchema, {
          enrollments: [makeEnrollment({ status: "pending" })],
          nextPageToken: "",
        }),
    });

    await screen.findByText("Ingeniería Civil");
    expect(
      screen.getByRole("button", { name: /pagar/i }),
    ).toBeInTheDocument();
  });

  it("does NOT show a Pagar button for paid rows", async () => {
    renderOwnEnrollments({
      listOwnEnrollments: async () =>
        create(ListEnrollmentsResponseSchema, {
          enrollments: [makeEnrollment({ status: "paid" })],
          nextPageToken: "",
        }),
    });

    await screen.findByText("Ingeniería Civil");
    expect(
      screen.queryByRole("button", { name: /pagar/i }),
    ).not.toBeInTheDocument();
  });

  it("does NOT show a Pagar button for cancelled rows", async () => {
    renderOwnEnrollments({
      listOwnEnrollments: async () =>
        create(ListEnrollmentsResponseSchema, {
          enrollments: [makeEnrollment({ status: "cancelled" })],
          nextPageToken: "",
        }),
    });

    await screen.findByText("Ingeniería Civil");
    expect(
      screen.queryByRole("button", { name: /pagar/i }),
    ).not.toBeInTheDocument();
  });

  it("clicking Pagar opens the PayOwnEnrollmentDialog", async () => {
    const user = userEvent.setup();
    renderOwnEnrollments({
      listOwnEnrollments: async () =>
        create(ListEnrollmentsResponseSchema, {
          enrollments: [makeEnrollment({ status: "pending" })],
          nextPageToken: "",
        }),
      markOwnEnrollmentPaid: async () => makeEnrollment({ status: "paid" }),
    });

    await user.click(await screen.findByRole("button", { name: /pagar/i }));

    // AlertDialog title should appear
    await screen.findByRole("alertdialog");
    await screen.findByText(/confirmar pago/i);
  });
});

describe("OwnEnrollmentsList — loading state", () => {
  it("shows skeleton with aria-busy while loading", async () => {
    renderOwnEnrollments({
      // biome-ignore lint/suspicious/noEmptyBlockStatements: intentional never-resolving promise
      listOwnEnrollments: () => new Promise(() => {}),
    });

    const skeleton = await screen.findByRole("status", {
      name: "Cargando tus matrículas",
    });
    expect(skeleton).toBeInTheDocument();
    expect(skeleton).toHaveAttribute("aria-busy", "true");
  });
});

describe("OwnEnrollmentsList — empty state", () => {
  it("shows empty state message when no enrollments exist", async () => {
    renderOwnEnrollments({
      listOwnEnrollments: async () =>
        create(ListEnrollmentsResponseSchema, {
          enrollments: [],
          nextPageToken: "",
        }),
    });

    await screen.findByText("Todavía no tienes matrículas.");
  });
});

describe("OwnEnrollmentsList — error state", () => {
  it("shows error message and Reintentar button on fetch failure", async () => {
    renderOwnEnrollments({
      listOwnEnrollments: async () => {
        throw new ConnectError("unavailable", Code.Unavailable);
      },
    });

    await screen.findByText("No se pudieron cargar tus matrículas.");
    expect(
      screen.getByRole("button", { name: /reintentar/i }),
    ).toBeInTheDocument();
  });

  it("Reintentar triggers a new listOwnEnrollments request", async () => {
    const user = userEvent.setup();
    let callCount = 0;

    renderOwnEnrollments({
      listOwnEnrollments: vi.fn(async () => {
        callCount++;
        if (callCount === 1)
          throw new ConnectError("unavailable", Code.Unavailable);
        return create(ListEnrollmentsResponseSchema, {
          enrollments: [makeEnrollment()],
          nextPageToken: "",
        });
      }),
    });

    await screen.findByRole("button", { name: /reintentar/i });
    await user.click(screen.getByRole("button", { name: /reintentar/i }));
    await screen.findByText("Ingeniería Civil");
  });
});

describe("OwnEnrollmentsList — Cargar más pagination", () => {
  it("no Cargar más when nextPageToken is empty", async () => {
    renderOwnEnrollments({
      listOwnEnrollments: async () =>
        create(ListEnrollmentsResponseSchema, {
          enrollments: [makeEnrollment()],
          nextPageToken: "",
        }),
    });

    await screen.findByText("Ingeniería Civil");
    expect(
      screen.queryByRole("button", { name: /cargar más/i }),
    ).not.toBeInTheDocument();
  });

  it("Cargar más appends rows; prior rows remain", async () => {
    const user = userEvent.setup();
    let callCount = 0;

    const enr1 = makeEnrollment({
      id: "enr-1",
      programName: "Ingeniería Civil",
      year: 2026,
    });
    const enr2 = makeEnrollment({
      id: "enr-2",
      programName: "Arquitectura",
      year: 2025,
    });

    renderOwnEnrollments({
      listOwnEnrollments: vi.fn(async () => {
        callCount++;
        if (callCount === 1) {
          return create(ListEnrollmentsResponseSchema, {
            enrollments: [enr1],
            nextPageToken: "cursor-2",
          });
        }
        return create(ListEnrollmentsResponseSchema, {
          enrollments: [enr2],
          nextPageToken: "",
        });
      }),
    });

    await screen.findByText("Ingeniería Civil");
    expect(
      screen.getByRole("button", { name: /cargar más/i }),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /cargar más/i }));

    await screen.findByText("Arquitectura");
    expect(screen.getByText("Ingeniería Civil")).toBeInTheDocument();
  });

  it("Cargar más failure shows toast, existing rows remain", async () => {
    const user = userEvent.setup();
    let callCount = 0;

    renderOwnEnrollments({
      listOwnEnrollments: vi.fn(async () => {
        callCount++;
        if (callCount === 1) {
          return create(ListEnrollmentsResponseSchema, {
            enrollments: [makeEnrollment({ programName: "Ingeniería Civil" })],
            nextPageToken: "cursor-2",
          });
        }
        throw new ConnectError("unavailable", Code.Unavailable);
      }),
    });

    await screen.findByText("Ingeniería Civil");
    await user.click(screen.getByRole("button", { name: /cargar más/i }));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith(
        "No se pudieron cargar más matrículas.",
      );
    });
    expect(screen.getByText("Ingeniería Civil")).toBeInTheDocument();
  });
});
