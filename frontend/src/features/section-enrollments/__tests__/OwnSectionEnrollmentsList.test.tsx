/**
 * OwnSectionEnrollmentsList component tests (student own-view).
 *
 * The list renders at /app/section-enrollments with `ownSectionEnrollmentsSearchSchema`
 * (pageSize only — read-only participant view, no filters).
 *
 * Covers:
 *  - Renders sectionId from wire.
 *  - Loading skeleton (role=status, aria-busy).
 *  - Empty state message.
 *  - Initial error state + Reintentar button.
 *  - Cargar más appends rows; prior rows remain.
 *  - No Cargar más when nextPageToken is empty.
 *  - Cargar más failure shows toast.
 *  - Calls listOwnSectionEnrollments, NOT listSectionEnrollments.
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import type { AuthenticatedSession } from "@/features/auth";
import {
  ListSectionEnrollmentsResponseSchema,
  SectionEnrollmentSchema,
  SectionEnrollmentService,
} from "@/gen/section_enrollment/v1/section_enrollment_pb";
import { renderWithProviders } from "@/test";

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

type SectionEnrollmentImpl = Partial<
  ServiceImpl<typeof SectionEnrollmentService>
>;

const studentSession = {
  status: "authenticated" as const,
  userId: "s1",
  email: "student@test.com",
  roles: ["student"],
  permissions: ["section_enrollment.view_own"],
};

const studentSessionSource = {
  getSession: async (): Promise<AuthenticatedSession> => ({
    userId: studentSession.userId,
    email: studentSession.email,
    roles: studentSession.roles,
    permissions: studentSession.permissions,
  }),
};

function makeSectionEnrollment(
  overrides: MessageInitShape<typeof SectionEnrollmentSchema> = {},
) {
  return create(SectionEnrollmentSchema, {
    id: "se-1",
    enrollmentId: "enr-1",
    sectionId: "sec-abc-123",
    status: "in_progress",
    registeredAt: "2026-01-15T00:00:00Z",
    createdAt: "2026-01-15T00:00:00Z",
    updatedAt: "2026-01-15T00:00:00Z",
    finalGrade: "",
    studentId: "s1",
    ...overrides,
  });
}

function renderOwnSectionEnrollments(
  handlers: SectionEnrollmentImpl,
  route = "/app/section-enrollments?pageSize=20",
) {
  return renderWithProviders({
    route,
    transport: makeStubTransport([SectionEnrollmentService, handlers]),
    session: studentSession,
    sessionSource: studentSessionSource,
  });
}

describe("OwnSectionEnrollmentsList — data display", () => {
  it("renders sectionId from wire in the table", async () => {
    renderOwnSectionEnrollments({
      listOwnSectionEnrollments: async () =>
        create(ListSectionEnrollmentsResponseSchema, {
          sectionEnrollments: [
            makeSectionEnrollment({ sectionId: "sec-abc-123" }),
          ],
          nextPageToken: "",
        }),
    });

    await screen.findByText("sec-abc-123");
  });

  it("renders the course label and period when the wire is enriched", async () => {
    renderOwnSectionEnrollments({
      listOwnSectionEnrollments: async () =>
        create(ListSectionEnrollmentsResponseSchema, {
          sectionEnrollments: [
            makeSectionEnrollment({
              courseName: "Cálculo I",
              courseCode: "MAT-101",
              periodYear: 2026,
              periodTerm: 1,
            }),
          ],
          nextPageToken: "",
        }),
    });

    await screen.findByText("Cálculo I (MAT-101)");
    expect(screen.getByText("2026 · Semestre 1")).toBeInTheDocument();
  });

  it("renders status for each row", async () => {
    renderOwnSectionEnrollments({
      listOwnSectionEnrollments: async () =>
        create(ListSectionEnrollmentsResponseSchema, {
          sectionEnrollments: [
            makeSectionEnrollment({ status: "in_progress" }),
          ],
          nextPageToken: "",
        }),
    });

    await screen.findByText("sec-abc-123");
    expect(screen.getByText("En curso")).toBeInTheDocument();
  });

  it("does NOT call listSectionEnrollments (admin RPC)", async () => {
    const adminRpc = vi.fn();
    renderOwnSectionEnrollments({
      listOwnSectionEnrollments: async () =>
        create(ListSectionEnrollmentsResponseSchema, {
          sectionEnrollments: [],
          nextPageToken: "",
        }),
      listSectionEnrollments: adminRpc,
    });

    await screen.findByText("Todavía no tienes inscripciones a secciones.");
    expect(adminRpc).not.toHaveBeenCalled();
  });
});

describe("OwnSectionEnrollmentsList — loading state", () => {
  it("shows skeleton with aria-busy while loading", async () => {
    renderOwnSectionEnrollments({
      // biome-ignore lint/suspicious/noEmptyBlockStatements: intentional never-resolving promise
      listOwnSectionEnrollments: () => new Promise(() => {}),
    });

    const skeleton = await screen.findByRole("status", {
      name: "Cargando tus inscripciones",
    });
    expect(skeleton).toBeInTheDocument();
    expect(skeleton).toHaveAttribute("aria-busy", "true");
  });
});

describe("OwnSectionEnrollmentsList — empty state", () => {
  it("shows empty state message when no section enrollments exist", async () => {
    renderOwnSectionEnrollments({
      listOwnSectionEnrollments: async () =>
        create(ListSectionEnrollmentsResponseSchema, {
          sectionEnrollments: [],
          nextPageToken: "",
        }),
    });

    await screen.findByText("Todavía no tienes inscripciones a secciones.");
  });
});

describe("OwnSectionEnrollmentsList — error state", () => {
  it("shows error message and Reintentar button on fetch failure", async () => {
    renderOwnSectionEnrollments({
      listOwnSectionEnrollments: async () => {
        throw new ConnectError("unavailable", Code.Unavailable);
      },
    });

    await screen.findByText("No se pudieron cargar tus inscripciones.");
    expect(
      screen.getByRole("button", { name: /reintentar/i }),
    ).toBeInTheDocument();
  });

  it("Reintentar triggers a new listOwnSectionEnrollments request", async () => {
    const user = userEvent.setup();
    let callCount = 0;

    renderOwnSectionEnrollments({
      listOwnSectionEnrollments: vi.fn(async () => {
        callCount++;
        if (callCount === 1)
          throw new ConnectError("unavailable", Code.Unavailable);
        return create(ListSectionEnrollmentsResponseSchema, {
          sectionEnrollments: [makeSectionEnrollment()],
          nextPageToken: "",
        });
      }),
    });

    await screen.findByRole("button", { name: /reintentar/i });
    await user.click(screen.getByRole("button", { name: /reintentar/i }));
    await screen.findByText("sec-abc-123");
  });
});

describe("OwnSectionEnrollmentsList — Cargar más pagination", () => {
  it("no Cargar más when nextPageToken is empty", async () => {
    renderOwnSectionEnrollments({
      listOwnSectionEnrollments: async () =>
        create(ListSectionEnrollmentsResponseSchema, {
          sectionEnrollments: [makeSectionEnrollment()],
          nextPageToken: "",
        }),
    });

    await screen.findByText("sec-abc-123");
    expect(
      screen.queryByRole("button", { name: /cargar más/i }),
    ).not.toBeInTheDocument();
  });

  it("Cargar más appends rows; prior rows remain", async () => {
    const user = userEvent.setup();
    let callCount = 0;

    const se1 = makeSectionEnrollment({
      id: "se-1",
      sectionId: "sec-aaa-111",
    });
    const se2 = makeSectionEnrollment({
      id: "se-2",
      sectionId: "sec-bbb-222",
    });

    renderOwnSectionEnrollments({
      listOwnSectionEnrollments: vi.fn(async () => {
        callCount++;
        if (callCount === 1) {
          return create(ListSectionEnrollmentsResponseSchema, {
            sectionEnrollments: [se1],
            nextPageToken: "cursor-2",
          });
        }
        return create(ListSectionEnrollmentsResponseSchema, {
          sectionEnrollments: [se2],
          nextPageToken: "",
        });
      }),
    });

    await screen.findByText("sec-aaa-111");
    expect(
      screen.getByRole("button", { name: /cargar más/i }),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /cargar más/i }));

    await screen.findByText("sec-bbb-222");
    expect(screen.getByText("sec-aaa-111")).toBeInTheDocument();
  });

  it("Cargar más failure shows toast, existing rows remain", async () => {
    const user = userEvent.setup();
    let callCount = 0;

    renderOwnSectionEnrollments({
      listOwnSectionEnrollments: vi.fn(async () => {
        callCount++;
        if (callCount === 1) {
          return create(ListSectionEnrollmentsResponseSchema, {
            sectionEnrollments: [
              makeSectionEnrollment({ sectionId: "sec-aaa-111" }),
            ],
            nextPageToken: "cursor-2",
          });
        }
        throw new ConnectError("unavailable", Code.Unavailable);
      }),
    });

    await screen.findByText("sec-aaa-111");
    await user.click(screen.getByRole("button", { name: /cargar más/i }));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith(
        "No se pudieron cargar más inscripciones.",
      );
    });
    expect(screen.getByText("sec-aaa-111")).toBeInTheDocument();
  });
});
