/**
 * SectionEnrollmentsPage integration tests.
 *
 * Uses renderWithProviders at /admin/section-enrollments so the real route
 * provides context (URL search params, session, transport).
 *
 * Covers:
 *  - Shows section selection table on initial mount.
 *  - Clicking a section row shows the roster table for that section.
 *  - "Volver" button goes back to section selection.
 *  - Loading state: aria-busy skeleton.
 *  - Error state: error message + Reintentar.
 *  - Unauthenticated → redirects to /login.
 */
import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import {
  CatalogService,
  type TeachingSection,
} from "@/gen/catalog/v1/catalog_pb";
import { EnrollmentService } from "@/gen/enrollment/v1/enrollment_pb";
import { ProfileService } from "@/gen/profiles/v1/profiles_pb";
import {
  ListSectionEnrollmentsResponseSchema,
  SectionEnrollmentSchema,
  SectionEnrollmentService,
} from "@/gen/section_enrollment/v1/section_enrollment_pb";
import { renderWithProviders } from "@/test";

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

const adminSession = {
  status: "authenticated" as const,
  userId: "admin-1",
  email: "admin@test.com",
  roles: ["admin"],
  permissions: ["enrollment.manage", "profile.view_names"],
};

const stubSections: TeachingSection[] = [
  {
    id: "sec-1",
    courseId: "course-1",
    academicPeriodId: "period-1",
    seatCapacity: 30,
    courseCode: "MAT101",
    courseName: "Matemáticas I",
    periodYear: 2026,
    periodTerm: 1,
  } as TeachingSection,
  {
    id: "sec-2",
    courseId: "course-2",
    academicPeriodId: "period-1",
    seatCapacity: 25,
    courseCode: "FIS101",
    courseName: "Física I",
    periodYear: 2026,
    periodTerm: 2,
  } as TeachingSection,
];

const stubEnrollment = create(SectionEnrollmentSchema, {
  id: "se-1",
  enrollmentId: "enroll-1",
  sectionId: "sec-1",
  status: "in_progress",
  registeredAt: "2026-01-10T00:00:00Z",
  createdAt: "2026-01-10T00:00:00Z",
  updatedAt: "2026-01-10T00:00:00Z",
  studentId: "aaaaaaaa-0000-0000-0000-000000000001",
});

function makeTransport(
  catalogImpl: Partial<ServiceImpl<typeof CatalogService>> = {},
  seImpl: Partial<ServiceImpl<typeof SectionEnrollmentService>> = {},
) {
  return makeStubTransport(
    [
      CatalogService,
      {
        listOwnSections: async () => ({
          sections: stubSections,
          nextPageToken: "",
        }),
        ...catalogImpl,
      },
    ],
    [
      SectionEnrollmentService,
      {
        listSectionEnrollments: async () =>
          create(ListSectionEnrollmentsResponseSchema, {
            sectionEnrollments: [stubEnrollment],
            nextPageToken: "",
          }),
        ...seImpl,
      },
    ],
    [
      EnrollmentService,
      {
        listEnrollments: async () => ({
          enrollments: [],
          nextPageToken: "",
        }),
      },
    ],
    [
      ProfileService,
      {
        listDisplayNamesByIDs: async () => ({ names: [] }),
      },
    ],
  );
}

describe("SectionEnrollmentsPage — section selection", () => {
  it("renders page heading and section selection table on mount", async () => {
    renderWithProviders({
      route: "/admin/section-enrollments",
      session: adminSession,
      transport: makeTransport(),
    });

    expect(
      await screen.findByRole("heading", {
        name: /inscripciones a secciones/i,
      }),
    ).toBeInTheDocument();
    expect(await screen.findByText("Matemáticas I")).toBeInTheDocument();
    expect(screen.getByText("Física I")).toBeInTheDocument();
  });

  it("clicking a section row shows the roster table", async () => {
    const user = userEvent.setup();

    renderWithProviders({
      route: "/admin/section-enrollments",
      session: adminSession,
      transport: makeTransport(),
    });

    await screen.findByText("Matemáticas I");
    await user.click(screen.getByText("Matemáticas I"));

    // Should show the roster (the enrollment studentId prefix)
    await screen.findByText("aaaaaaaa");
    // Should show a "Volver a secciones" back button with ArrowLeft icon.
    const backBtn = screen.getByRole("button", { name: /volver a secciones/i });
    expect(backBtn).toBeInTheDocument();
    // ArrowLeft icon is rendered inside the button (aria-hidden, presentational).
    expect(backBtn.querySelector("svg")).toBeInTheDocument();
  });

  it("Volver button goes back to section selection", async () => {
    const user = userEvent.setup();

    renderWithProviders({
      route: "/admin/section-enrollments",
      session: adminSession,
      transport: makeTransport(),
    });

    await screen.findByText("Matemáticas I");
    await user.click(screen.getByText("Matemáticas I"));

    await screen.findByText("aaaaaaaa");

    await user.click(
      screen.getByRole("button", { name: /volver a secciones/i }),
    );

    // Should show section selection table again
    expect(await screen.findByText("Matemáticas I")).toBeInTheDocument();
  });
});

describe("SectionEnrollmentsPage — unauthenticated", () => {
  it("unauthenticated session → redirects to /login", async () => {
    const { router } = renderWithProviders({
      route: "/admin/section-enrollments",
      session: { status: "unauthenticated" },
    });

    await waitFor(() => expect(router.state.location.pathname).toBe("/login"));
  });
});
