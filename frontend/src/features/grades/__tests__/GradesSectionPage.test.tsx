import { create } from "@bufbuild/protobuf";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { makeStubTransport } from "@/core/test";
import type { Permission, SessionState } from "@/features/auth";
import {
  CatalogService,
  TeachingSectionSchema,
} from "@/gen/catalog/v1/catalog_pb";
import { GradesService } from "@/gen/grades/v1/grades_pb";
import { ProfileService } from "@/gen/profiles/v1/profiles_pb";
import { SectionEnrollmentService } from "@/gen/section_enrollment/v1/section_enrollment_pb";
import { renderWithProviders } from "@/test";

// ──────────────────────────────────────────────
// Fixtures
// ──────────────────────────────────────────────

const stubSection = create(TeachingSectionSchema, {
  id: "sec-99",
  courseId: "course-99",
  academicPeriodId: "period-1",
  seatCapacity: 30,
  courseCode: "PROG101",
  courseName: "Programación 1",
  periodYear: 2024,
  periodTerm: 1,
});

function session(permissions: Permission[]): SessionState {
  return {
    status: "authenticated",
    userId: "u-1",
    email: "teacher@test.com",
    roles: ["teacher"],
    permissions,
  };
}

/**
 * Minimal stub that satisfies all RPCs called by GradeRecordingGrid's
 * composition hooks at mount time.
 */
const minimalGridTransport = makeStubTransport(
  [
    CatalogService,
    {
      listOwnSections: async () => ({
        sections: [stubSection],
        nextPageToken: "",
      }),
      listCourses: async () => ({ courses: [], nextPageToken: "" }),
    },
  ],
  [GradesService, { listEvaluations: async () => ({ evaluations: [] }) }],
  [
    SectionEnrollmentService,
    {
      listSectionRosterForTeacher: async () => ({
        sectionEnrollments: [],
        nextPageToken: "",
      }),
    },
  ],
  [ProfileService, { listDisplayNamesByIDs: async () => ({ names: [] }) }],
);

/** Transport that returns no sections — simulates a not-found deep-link. */
const emptySectionsTransport = makeStubTransport(
  [
    CatalogService,
    {
      listOwnSections: async () => ({ sections: [], nextPageToken: "" }),
      listCourses: async () => ({ courses: [], nextPageToken: "" }),
    },
  ],
  [GradesService, { listEvaluations: async () => ({ evaluations: [] }) }],
  [
    SectionEnrollmentService,
    {
      listSectionRosterForTeacher: async () => ({
        sectionEnrollments: [],
        nextPageToken: "",
      }),
    },
  ],
  [ProfileService, { listDisplayNamesByIDs: async () => ({ names: [] }) }],
);

// ──────────────────────────────────────────────
// R-01: Table view at /admin/grades
// ──────────────────────────────────────────────

describe("grades sub-route navigation", () => {
  it("R-01: /admin/grades shows the section selection table, not the grid", async () => {
    renderWithProviders({
      route: "/admin/grades",
      session: session(["grades.write"]),
      transport: minimalGridTransport,
    });

    // Use heading role to disambiguate from the sidebar nav link
    expect(
      await screen.findByRole("heading", { name: "Notas" }),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Selecciona una sección para registrar notas."),
    ).toBeInTheDocument();
  });

  it("R-02: /admin/grades/$sectionId resolves section by id and renders the grid when section exists", async () => {
    renderWithProviders({
      route: "/admin/grades/sec-99",
      session: session(["grades.write"]),
      transport: minimalGridTransport,
    });

    // GradeRecordingGrid shows the course name once section is resolved
    expect(await screen.findByText("Programación 1")).toBeInTheDocument();
    // "Volver a secciones" back button is present
    expect(
      screen.getByRole("button", { name: /volver a secciones/i }),
    ).toBeInTheDocument();
  });

  it("R-03: /admin/grades/$sectionId shows not-found message when section is not in the user's list", async () => {
    renderWithProviders({
      route: "/admin/grades/unknown-section-id",
      session: session(["grades.write"]),
      transport: emptySectionsTransport,
    });

    expect(
      await screen.findByText("No se pudo cargar la sección."),
    ).toBeInTheDocument();
    // A link back to /admin/grades is present with the canonical back-link style.
    const backLink = screen.getByRole("link", { name: /volver a secciones/i });
    expect(backLink).toBeInTheDocument();
    // ArrowLeft icon is rendered inside the link (aria-hidden, presentational).
    expect(backLink.querySelector("svg")).toBeInTheDocument();
  });

  it("R-04: the grid renders a 'Volver a secciones' back button that triggers navigation", async () => {
    const user = userEvent.setup();

    const { router } = renderWithProviders({
      route: "/admin/grades/sec-99",
      session: session(["grades.write"]),
      transport: minimalGridTransport,
    });

    const backButton = await screen.findByRole("button", {
      name: /volver a secciones/i,
    });
    // The button is wired to navigate({ to: "/admin/grades" }); clicking it
    // triggers a navigation event (the router leaves /admin/grades/sec-99).
    await user.click(backButton);

    await waitFor(() =>
      expect(router.state.location.pathname).not.toBe("/admin/grades/sec-99"),
    );
  });

  it("R-05: /admin/grades/$sectionId redirects a zero-eligibility session to /forbidden", async () => {
    // A session with no permissions has no area eligibility; admin/route.tsx
    // redirects to /forbidden (not /app, since participant eligibility is also absent).
    const { router } = renderWithProviders({
      route: "/admin/grades/sec-99",
      session: session([]),
    });

    await waitFor(() =>
      expect(router.state.location.pathname).toBe("/forbidden"),
    );
  });
});

// ──────────────────────────────────────────────
// R-06: Scheme pre-scope (Change 2)
// ──────────────────────────────────────────────

describe("AdminSchemeButton — scheme pre-scope", () => {
  it("R-06: renders GradeRecordingGrid with AdminSchemeButton for grades.override session", async () => {
    renderWithProviders({
      route: "/admin/grades/sec-99",
      session: session(["grades.override", "grades.read"]),
      transport: minimalGridTransport,
    });

    // The grid header renders for admin with the AdminSchemeButton
    expect(await screen.findByText("Programación 1")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /administrar notas/i }),
    ).toBeInTheDocument();
  });
});
