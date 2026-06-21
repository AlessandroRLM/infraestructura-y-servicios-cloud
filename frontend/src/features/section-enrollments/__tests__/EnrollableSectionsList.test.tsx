/**
 * EnrollableSectionsList component tests (student self-enroll view).
 *
 * Covers:
 *  - Renders course/period/seats from wire.
 *  - Loading skeleton (role=status, aria-busy).
 *  - Empty state message.
 *  - Initial error state + Reintentar button.
 *  - Clicking "Inscribirme" calls enrollOwnSection with correct { sectionId, programId }.
 *  - Success path: toast shown after enroll.
 *  - Error path: toast shown with friendly message (no raw Code.*).
 *  - Cargar más appends rows; prior rows remain.
 *  - No Cargar más when nextPageToken is empty.
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
  EnrollableSectionSchema,
  ListEnrollableSectionsResponseSchema,
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
  permissions: ["sections.enroll", "section_enrollment.view_own"],
};

const studentSessionSource = {
  getSession: async (): Promise<AuthenticatedSession> => ({
    userId: studentSession.userId,
    email: studentSession.email,
    roles: studentSession.roles,
    permissions: studentSession.permissions,
  }),
};

function makeEnrollableSection(
  overrides: MessageInitShape<typeof EnrollableSectionSchema> = {},
) {
  return create(EnrollableSectionSchema, {
    sectionId: "sec-abc-123",
    programId: "prog-1",
    courseName: "Cálculo I",
    courseCode: "MAT-101",
    periodYear: 2026,
    periodTerm: 1,
    seatsAvailable: 5,
    ...overrides,
  });
}

function renderEnrollableSections(
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

describe("EnrollableSectionsList — data display", () => {
  it("renders course name and code, period, and seats", async () => {
    renderEnrollableSections({
      listEnrollableSections: async () =>
        create(ListEnrollableSectionsResponseSchema, {
          sections: [
            makeEnrollableSection({
              courseName: "Cálculo I",
              courseCode: "MAT-101",
              periodYear: 2026,
              periodTerm: 1,
              seatsAvailable: 5,
            }),
          ],
          nextPageToken: "",
        }),
      listOwnSectionEnrollments: async () => ({
        sectionEnrollments: [],
        nextPageToken: "",
      }),
    });

    await screen.findByText("Cálculo I (MAT-101)");
    expect(screen.getByText("2026 · Semestre 1")).toBeInTheDocument();
    expect(screen.getByText("5")).toBeInTheDocument();
  });
});

describe("EnrollableSectionsList — loading state", () => {
  it("shows skeleton with aria-busy while loading", async () => {
    renderEnrollableSections({
      // biome-ignore lint/suspicious/noEmptyBlockStatements: intentional never-resolving promise
      listEnrollableSections: () => new Promise(() => {}),
      listOwnSectionEnrollments: async () => ({
        sectionEnrollments: [],
        nextPageToken: "",
      }),
    });

    const skeleton = await screen.findByRole("status", {
      name: "Cargando secciones disponibles",
    });
    expect(skeleton).toBeInTheDocument();
    expect(skeleton).toHaveAttribute("aria-busy", "true");
  });
});

describe("EnrollableSectionsList — empty state", () => {
  it("shows empty state message when no enrollable sections exist", async () => {
    renderEnrollableSections({
      listEnrollableSections: async () =>
        create(ListEnrollableSectionsResponseSchema, {
          sections: [],
          nextPageToken: "",
        }),
      listOwnSectionEnrollments: async () => ({
        sectionEnrollments: [],
        nextPageToken: "",
      }),
    });

    await screen.findByText("No hay secciones disponibles para inscribirte.");
  });
});

describe("EnrollableSectionsList — error state", () => {
  it("shows error message and Reintentar button on fetch failure", async () => {
    renderEnrollableSections({
      listEnrollableSections: async () => {
        throw new ConnectError("unavailable", Code.Unavailable);
      },
      listOwnSectionEnrollments: async () => ({
        sectionEnrollments: [],
        nextPageToken: "",
      }),
    });

    await screen.findByText("No se pudieron cargar las secciones disponibles.");
    expect(
      screen.getByRole("button", { name: /reintentar/i }),
    ).toBeInTheDocument();
  });
});

describe("EnrollableSectionsList — enroll action", () => {
  it("clicking Inscribirme calls enrollOwnSection with correct sectionId and programId", async () => {
    const user = userEvent.setup();
    const enrollOwnSection = vi.fn(async () =>
      create(SectionEnrollmentSchema, {
        id: "se-new",
        enrollmentId: "enroll-1",
        sectionId: "sec-abc-123",
        status: "in_progress",
        registeredAt: "2026-01-15T00:00:00Z",
        createdAt: "2026-01-15T00:00:00Z",
        updatedAt: "2026-01-15T00:00:00Z",
      }),
    );

    renderEnrollableSections({
      listEnrollableSections: async () =>
        create(ListEnrollableSectionsResponseSchema, {
          sections: [
            makeEnrollableSection({
              sectionId: "sec-abc-123",
              programId: "prog-1",
            }),
          ],
          nextPageToken: "",
        }),
      listOwnSectionEnrollments: async () => ({
        sectionEnrollments: [],
        nextPageToken: "",
      }),
      enrollOwnSection,
    });

    await screen.findByRole("button", { name: /inscribirme/i });
    await user.click(screen.getByRole("button", { name: /inscribirme/i }));

    await waitFor(() =>
      expect(enrollOwnSection).toHaveBeenCalledWith(
        expect.objectContaining({
          sectionId: "sec-abc-123",
          programId: "prog-1",
        }),
        expect.anything(),
      ),
    );
  });

  it("success path shows a success toast", async () => {
    const user = userEvent.setup();

    renderEnrollableSections({
      listEnrollableSections: async () =>
        create(ListEnrollableSectionsResponseSchema, {
          sections: [makeEnrollableSection()],
          nextPageToken: "",
        }),
      listOwnSectionEnrollments: async () => ({
        sectionEnrollments: [],
        nextPageToken: "",
      }),
      enrollOwnSection: async () =>
        create(SectionEnrollmentSchema, {
          id: "se-new",
          enrollmentId: "enroll-1",
          sectionId: "sec-abc-123",
          status: "in_progress",
          registeredAt: "2026-01-15T00:00:00Z",
          createdAt: "2026-01-15T00:00:00Z",
          updatedAt: "2026-01-15T00:00:00Z",
        }),
    });

    await screen.findByRole("button", { name: /inscribirme/i });
    await user.click(screen.getByRole("button", { name: /inscribirme/i }));

    await waitFor(() => expect(toast.success).toHaveBeenCalled());
  });

  it("error path shows a friendly toast — no raw Code.* text in DOM", async () => {
    const user = userEvent.setup();

    renderEnrollableSections({
      listEnrollableSections: async () =>
        create(ListEnrollableSectionsResponseSchema, {
          sections: [makeEnrollableSection()],
          nextPageToken: "",
        }),
      listOwnSectionEnrollments: async () => ({
        sectionEnrollments: [],
        nextPageToken: "",
      }),
      enrollOwnSection: async () => {
        throw new ConnectError("window closed", Code.FailedPrecondition);
      },
    });

    await screen.findByRole("button", { name: /inscribirme/i });
    await user.click(screen.getByRole("button", { name: /inscribirme/i }));

    await waitFor(() => expect(toast.error).toHaveBeenCalled());
    // No raw code names or service names in the DOM
    expect(screen.queryByText(/FailedPrecondition/)).not.toBeInTheDocument();
    expect(
      screen.queryByText(/SectionEnrollmentService/),
    ).not.toBeInTheDocument();
  });
});

describe("EnrollableSectionsList — Cargar más pagination", () => {
  it("no Cargar más when nextPageToken is empty", async () => {
    renderEnrollableSections({
      listEnrollableSections: async () =>
        create(ListEnrollableSectionsResponseSchema, {
          sections: [makeEnrollableSection()],
          nextPageToken: "",
        }),
      listOwnSectionEnrollments: async () => ({
        sectionEnrollments: [],
        nextPageToken: "",
      }),
    });

    await screen.findByText("Cálculo I (MAT-101)");
    expect(
      screen.queryByRole("button", { name: /cargar más/i }),
    ).not.toBeInTheDocument();
  });

  it("Cargar más appends rows; prior rows remain", async () => {
    const user = userEvent.setup();
    let callCount = 0;

    const sec1 = makeEnrollableSection({
      sectionId: "sec-aaa-111",
      courseName: "Cálculo I",
      courseCode: "MAT-101",
    });
    const sec2 = makeEnrollableSection({
      sectionId: "sec-bbb-222",
      courseName: "Física I",
      courseCode: "FIS-101",
    });

    renderEnrollableSections({
      listEnrollableSections: vi.fn(async () => {
        callCount++;
        if (callCount === 1) {
          return create(ListEnrollableSectionsResponseSchema, {
            sections: [sec1],
            nextPageToken: "cursor-2",
          });
        }
        return create(ListEnrollableSectionsResponseSchema, {
          sections: [sec2],
          nextPageToken: "",
        });
      }),
      listOwnSectionEnrollments: async () => ({
        sectionEnrollments: [],
        nextPageToken: "",
      }),
    });

    await screen.findByText("Cálculo I (MAT-101)");
    expect(
      screen.getByRole("button", { name: /cargar más/i }),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /cargar más/i }));

    await screen.findByText("Física I (FIS-101)");
    expect(screen.getByText("Cálculo I (MAT-101)")).toBeInTheDocument();
  });
});
