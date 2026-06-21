/**
 * SectionEnrollmentsTable component tests (admin section roster view).
 *
 * Covers:
 *  - Loading skeleton (aria-busy).
 *  - Populated rows with status, studentId prefix, registeredAt date.
 *  - Empty state when no enrollments.
 *  - Initial transport error + Reintentar.
 *  - Cargar más appends rows; prior rows remain.
 *  - No Cargar más when nextPageToken empty.
 *  - Retirar button visible for in_progress; absent for withdrawn.
 *  - Inscribir alumno button visible when canManage.
 *  - Resolved display name shown when ProfileService returns a name.
 *  - Falls back to studentId[:8] when ProfileService returns no match.
 */
import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import { EnrollmentService } from "@/gen/enrollment/v1/enrollment_pb";
import { ProfileService } from "@/gen/profiles/v1/profiles_pb";
import {
  ListSectionEnrollmentsResponseSchema,
  SectionEnrollmentSchema,
  SectionEnrollmentService,
} from "@/gen/section_enrollment/v1/section_enrollment_pb";
import { renderComponent } from "@/test";
import { SectionEnrollmentsTable } from "../components/SectionEnrollmentsTable";

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

type SectionEnrollmentImpl = Partial<
  ServiceImpl<typeof SectionEnrollmentService>
>;
type ProfileImpl = Partial<ServiceImpl<typeof ProfileService>>;

const adminSession = {
  status: "authenticated" as const,
  userId: "admin-1",
  email: "admin@test.com",
  roles: ["admin"],
  permissions: ["enrollment.manage", "profile.view_names"],
};

const inProgressEnrollment = create(SectionEnrollmentSchema, {
  id: "se-1",
  enrollmentId: "enroll-1",
  sectionId: "sec-1",
  status: "in_progress",
  registeredAt: "2026-01-10T00:00:00Z",
  createdAt: "2026-01-10T00:00:00Z",
  updatedAt: "2026-01-10T00:00:00Z",
  studentId: "aaaaaaaa-0000-0000-0000-000000000001",
});

const withdrawnEnrollment = create(SectionEnrollmentSchema, {
  id: "se-2",
  enrollmentId: "enroll-2",
  sectionId: "sec-1",
  status: "withdrawn",
  registeredAt: "2026-01-15T00:00:00Z",
  createdAt: "2026-01-15T00:00:00Z",
  updatedAt: "2026-01-20T00:00:00Z",
  studentId: "bbbbbbbb-0000-0000-0000-000000000002",
});

function renderTable(
  impl: SectionEnrollmentImpl,
  sectionId = "sec-1",
  session = adminSession,
  profileImpl: ProfileImpl = {
    listDisplayNamesByIDs: async () => ({ names: [] }),
  },
) {
  renderComponent(<SectionEnrollmentsTable sectionId={sectionId} />, {
    transport: makeStubTransport(
      [SectionEnrollmentService, impl],
      [
        EnrollmentService,
        {
          listEnrollments: async () => ({ enrollments: [], nextPageToken: "" }),
        },
      ],
      [ProfileService, profileImpl],
    ),
    session,
  });
}

describe("SectionEnrollmentsTable — loading", () => {
  it("shows aria-busy skeleton while listSectionEnrollments is pending", async () => {
    // biome-ignore lint/suspicious/noEmptyBlockStatements: intentional never-resolving promise for loading state test
    renderTable({ listSectionEnrollments: () => new Promise(() => {}) });

    const skeleton = await screen.findByRole("status", {
      name: "Cargando inscripciones",
    });
    expect(skeleton).toBeInTheDocument();
    expect(skeleton).toHaveAttribute("aria-busy", "true");
  });
});

describe("SectionEnrollmentsTable — populated rows", () => {
  it("shows studentId prefix and status for in_progress enrollment", async () => {
    renderTable({
      listSectionEnrollments: async () =>
        create(ListSectionEnrollmentsResponseSchema, {
          sectionEnrollments: [inProgressEnrollment],
          nextPageToken: "",
        }),
    });

    // studentId[:8] = "aaaaaaaa"
    await screen.findByText("aaaaaaaa");
    // Status
    expect(screen.getByText("En curso")).toBeInTheDocument();
  });

  it("shows Retirar button for in_progress enrollment", async () => {
    renderTable({
      listSectionEnrollments: async () =>
        create(ListSectionEnrollmentsResponseSchema, {
          sectionEnrollments: [inProgressEnrollment],
          nextPageToken: "",
        }),
    });

    await screen.findByText("aaaaaaaa");
    expect(
      screen.getByRole("button", { name: /retirar/i }),
    ).toBeInTheDocument();
  });

  it("does not show Retirar button for withdrawn enrollment", async () => {
    renderTable({
      listSectionEnrollments: async () =>
        create(ListSectionEnrollmentsResponseSchema, {
          sectionEnrollments: [withdrawnEnrollment],
          nextPageToken: "",
        }),
    });

    await screen.findByText("bbbbbbbb");
    expect(
      screen.queryByRole("button", { name: /retirar/i }),
    ).not.toBeInTheDocument();
  });
});

describe("SectionEnrollmentsTable — empty state", () => {
  it("shows empty copy when no enrollments", async () => {
    renderTable({
      listSectionEnrollments: async () =>
        create(ListSectionEnrollmentsResponseSchema, {
          sectionEnrollments: [],
          nextPageToken: "",
        }),
    });

    await screen.findByText(/no hay inscripciones/i);
  });
});

describe("SectionEnrollmentsTable — error state", () => {
  it("shows error message and Reintentar; no raw error code", async () => {
    renderTable({
      listSectionEnrollments: async () => {
        throw new ConnectError("unavailable", Code.Unavailable);
      },
    });

    await screen.findByText(/no se pudo cargar/i);
    expect(
      screen.getByRole("button", { name: /reintentar/i }),
    ).toBeInTheDocument();
    expect(screen.queryByText(/Unavailable/i)).not.toBeInTheDocument();
  });

  it("clicking Reintentar refetches and renders rows on success", async () => {
    const user = userEvent.setup();
    let callCount = 0;
    const listSectionEnrollments = vi.fn(async () => {
      callCount++;
      if (callCount === 1) {
        throw new ConnectError("unavailable", Code.Unavailable);
      }
      return create(ListSectionEnrollmentsResponseSchema, {
        sectionEnrollments: [inProgressEnrollment],
        nextPageToken: "",
      });
    });

    renderTable({ listSectionEnrollments });

    await screen.findByRole("button", { name: /reintentar/i });
    await user.click(screen.getByRole("button", { name: /reintentar/i }));

    await screen.findByText("aaaaaaaa");
    expect(listSectionEnrollments).toHaveBeenCalledTimes(2);
  });
});

describe("SectionEnrollmentsTable — pagination", () => {
  it("Cargar más appends rows; prior rows remain", async () => {
    const user = userEvent.setup();
    let callCount = 0;
    const listSectionEnrollments = vi.fn(async () => {
      callCount++;
      if (callCount === 1) {
        return create(ListSectionEnrollmentsResponseSchema, {
          sectionEnrollments: [inProgressEnrollment],
          nextPageToken: "cursor-page-2",
        });
      }
      return create(ListSectionEnrollmentsResponseSchema, {
        sectionEnrollments: [withdrawnEnrollment],
        nextPageToken: "",
      });
    });

    renderTable({ listSectionEnrollments });

    await screen.findByText("aaaaaaaa");
    expect(
      screen.getByRole("button", { name: /cargar más/i }),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /cargar más/i }));

    await screen.findByText("bbbbbbbb");
    expect(screen.getByText("aaaaaaaa")).toBeInTheDocument();
  });

  it("no Cargar más when nextPageToken is empty", async () => {
    renderTable({
      listSectionEnrollments: async () =>
        create(ListSectionEnrollmentsResponseSchema, {
          sectionEnrollments: [inProgressEnrollment],
          nextPageToken: "",
        }),
    });

    await screen.findByText("aaaaaaaa");
    expect(
      screen.queryByRole("button", { name: /cargar más/i }),
    ).not.toBeInTheDocument();
  });
});

describe("SectionEnrollmentsTable — Inscribir alumno button", () => {
  it("shows 'Inscribir alumno' button when canManage", async () => {
    renderTable({
      listSectionEnrollments: async () =>
        create(ListSectionEnrollmentsResponseSchema, {
          sectionEnrollments: [],
          nextPageToken: "",
        }),
    });

    await screen.findByText(/no hay inscripciones/i);
    expect(
      screen.getByRole("button", { name: /inscribir alumno/i }),
    ).toBeInTheDocument();
  });

  it("does not show 'Inscribir alumno' button when no manage permission", async () => {
    renderTable(
      {
        listSectionEnrollments: async () =>
          create(ListSectionEnrollmentsResponseSchema, {
            sectionEnrollments: [],
            nextPageToken: "",
          }),
      },
      "sec-1",
      {
        status: "authenticated",
        userId: "user-1",
        email: "user@test.com",
        roles: ["student"],
        permissions: [],
      },
    );

    await screen.findByText(/no hay inscripciones/i);
    expect(
      screen.queryByRole("button", { name: /inscribir alumno/i }),
    ).not.toBeInTheDocument();
  });
});

describe("SectionEnrollmentsTable — display name resolution", () => {
  it("shows resolved display name when ProfileService returns a match", async () => {
    renderTable(
      {
        listSectionEnrollments: async () =>
          create(ListSectionEnrollmentsResponseSchema, {
            sectionEnrollments: [inProgressEnrollment],
            nextPageToken: "",
          }),
      },
      "sec-1",
      adminSession,
      {
        listDisplayNamesByIDs: async () => ({
          names: [
            {
              userId: "aaaaaaaa-0000-0000-0000-000000000001",
              givenNames: "María",
              lastNamePaternal: "González",
            },
          ],
        }),
      },
    );

    // The resolved name should appear instead of the UUID prefix.
    expect(await screen.findByText("María González")).toBeInTheDocument();
    expect(screen.queryByText("aaaaaaaa")).not.toBeInTheDocument();
  });

  it("falls back to studentId[:8] when ProfileService returns no entry for the student", async () => {
    renderTable(
      {
        listSectionEnrollments: async () =>
          create(ListSectionEnrollmentsResponseSchema, {
            sectionEnrollments: [inProgressEnrollment],
            nextPageToken: "",
          }),
      },
      "sec-1",
      adminSession,
      {
        // Returns empty names — simulates permission gate not met or student not found.
        listDisplayNamesByIDs: async () => ({ names: [] }),
      },
    );

    // Fallback: studentId[:8] = "aaaaaaaa".
    await screen.findByText("aaaaaaaa");
    expect(screen.queryByText("María González")).not.toBeInTheDocument();
  });
});
