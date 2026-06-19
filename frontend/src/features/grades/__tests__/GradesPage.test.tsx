import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { makeStubTransport } from "@/core/test";
import type { Permission, SessionState } from "@/features/auth";
import { CatalogService } from "@/gen/catalog/v1/catalog_pb";
import { GradesService } from "@/gen/grades/v1/grades_pb";
import { ProfileService } from "@/gen/profiles/v1/profiles_pb";
import { SectionEnrollmentService } from "@/gen/section_enrollment/v1/section_enrollment_pb";
import { renderComponent } from "@/test";
import { GradesPage } from "../components/GradesPage";

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

describe("GradesPage — rewired recording flow (T20)", () => {
  it("S-01a: grades.write → renders SectionSelectionTable heading", async () => {
    renderComponent(<GradesPage />, {
      session: session(["grades.write"]),
      transport: minimalTransport,
    });

    // GradesPage renders the "Notas" heading + selection subtitle
    expect(screen.getByText("Notas")).toBeInTheDocument();
    expect(
      screen.getByText("Selecciona una sección para registrar notas."),
    ).toBeInTheDocument();
  });

  it("S-01b: grades.override → renders SectionSelectionTable heading (recording flow)", async () => {
    renderComponent(<GradesPage />, {
      session: session(["grades.override"]),
      transport: minimalTransport,
    });

    expect(screen.getByText("Notas")).toBeInTheDocument();
    expect(
      screen.getByText("Selecciona una sección para registrar notas."),
    ).toBeInTheDocument();
  });

  it("S-01c: no relevant permission → renders access denied message", () => {
    renderComponent(<GradesPage />, {
      session: session(["grades.read"]),
    });

    expect(screen.getByText("Notas")).toBeInTheDocument();
    expect(
      screen.getByText("No tienes permisos para acceder a esta sección."),
    ).toBeInTheDocument();
  });

  it("loading session → renders access denied (hasPermission returns false during loading)", () => {
    renderComponent(<GradesPage />, {
      session: { status: "loading" },
    });

    // hasPermission returns false during loading → no write/override
    expect(
      screen.getByText("No tienes permisos para acceder a esta sección."),
    ).toBeInTheDocument();
  });

  it("unauthenticated session → renders access denied", () => {
    renderComponent(<GradesPage />, {
      session: { status: "unauthenticated" },
    });

    expect(
      screen.getByText("No tienes permisos para acceder a esta sección."),
    ).toBeInTheDocument();
  });

  it("S-01d: grades.write → SchemeManagementView subtitle NOT in DOM (recording flow, no scheme admin)", () => {
    renderComponent(<GradesPage />, {
      session: session(["grades.write"]),
      transport: minimalTransport,
    });

    // SchemeManagementView should NOT be rendered directly
    expect(
      screen.queryByText(
        "Administra los esquemas de evaluación por asignatura.",
      ),
    ).not.toBeInTheDocument();
  });
});
