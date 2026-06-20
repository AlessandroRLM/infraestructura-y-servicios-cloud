import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { makeStubTransport } from "@/core/test";
import type { Permission, SessionState } from "@/features/auth";
import { CatalogService } from "@/gen/catalog/v1/catalog_pb";
import { GradesService } from "@/gen/grades/v1/grades_pb";
import { ProfileService } from "@/gen/profiles/v1/profiles_pb";
import { SectionEnrollmentService } from "@/gen/section_enrollment/v1/section_enrollment_pb";
import { renderWithProviders } from "@/test";

/**
 * Full transport stub covering all services queried at mount time:
 * - CatalogService.listOwnSections: used by SectionSelectionTable via useOwnSections.
 * - CatalogService.listCourses: used by SchemeManagementView if admin clicks "Administrar Notas".
 * - GradesService.listEvaluations: used by useSectionGrid if a section is selected.
 * - SectionEnrollmentService.listSectionRosterForTeacher: used by useSectionGrid.
 * - ProfileService.listDisplayNamesByIDs: used by useSectionGrid for display names.
 */
const minimalTransport = makeStubTransport(
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

function session(permissions: Permission[]): SessionState {
  return {
    status: "authenticated",
    userId: "u-1",
    email: "admin@test.com",
    roles: ["admin"],
    permissions,
  };
}

// GradesPage now uses useNavigate, so it must run inside a RouterProvider.
// renderWithProviders starts at a specific route and renders the full routeTree.
// Note: the admin sidebar also renders a "Notas" nav link — use heading role to
// disambiguate from the sidebar link.
describe("GradesPage — rewired recording flow (T20)", () => {
  it("S-01a: grades.write → renders SectionSelectionTable heading", async () => {
    renderWithProviders({
      route: "/admin/grades",
      session: session(["grades.write"]),
      transport: minimalTransport,
    });

    // Find the page heading (h1), not the sidebar link
    expect(
      await screen.findByRole("heading", { name: "Notas" }),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Selecciona una sección para registrar notas."),
    ).toBeInTheDocument();
  });

  it("S-01b: grades.override → renders SectionSelectionTable heading (recording flow)", async () => {
    renderWithProviders({
      route: "/admin/grades",
      session: session(["grades.override", "grades.read"]),
      transport: minimalTransport,
    });

    expect(
      await screen.findByRole("heading", { name: "Notas" }),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Selecciona una sección para registrar notas."),
    ).toBeInTheDocument();
  });

  it("S-01c: no relevant permission → renders access denied message (grades.read only)", async () => {
    // grades.read alone passes the route guard (grades.read is in ROUTE_PERMISSIONS[/admin/grades])
    // but GradesPage itself checks grades.write OR grades.override — neither satisfied.
    renderWithProviders({
      route: "/admin/grades",
      session: session(["grades.read"]),
      transport: minimalTransport,
    });

    expect(
      await screen.findByRole("heading", { name: "Notas" }),
    ).toBeInTheDocument();
    expect(
      await screen.findByText(
        "No tienes permisos para acceder a esta sección.",
      ),
    ).toBeInTheDocument();
  });

  it("loading session → redirects away from /admin/grades (no area eligibility)", async () => {
    // Loading session → no permissions → area guard redirects from /admin
    const { router } = renderWithProviders({
      route: "/admin/grades",
      session: { status: "loading" },
    });

    await waitFor(() =>
      expect(router.state.location.pathname).not.toBe("/admin/grades"),
    );
  });

  it("unauthenticated session → redirects to /login", async () => {
    const { router } = renderWithProviders({
      route: "/admin/grades",
      session: { status: "unauthenticated" },
    });

    await waitFor(() => expect(router.state.location.pathname).toBe("/login"));
  });

  it("S-01d: grades.write → SchemeManagementView subtitle NOT in DOM (recording flow, no scheme admin)", async () => {
    renderWithProviders({
      route: "/admin/grades",
      session: session(["grades.write"]),
      transport: minimalTransport,
    });

    await screen.findByRole("heading", { name: "Notas" });
    // SchemeManagementView should NOT be rendered directly
    expect(
      screen.queryByText(
        "Administra los esquemas de evaluación por asignatura.",
      ),
    ).not.toBeInTheDocument();
  });
});
