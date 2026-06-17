import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { makeStubTransport } from "@/core/test";
import type { Permission, SessionState } from "@/features/auth";
import { CatalogService } from "@/gen/catalog/v1/catalog_pb";
import { GradesService } from "@/gen/grades/v1/grades_pb";
import { renderComponent } from "@/test";
import { GradesPage } from "../components/GradesPage";

/**
 * Minimal transport stub covering every service queried at mount time.
 * CatalogService.listCourses: satisfies useCourses inside SchemeManagementView.
 * GradesService.listEvaluations: keeps the stub resilient if SchemeManagementView
 * ever pre-selects a course at mount; absent it, an evaluation query would surface
 * as an unhandled transport error instead of the intended assertion failure.
 */
const minimalTransport = makeStubTransport(
  [
    CatalogService,
    { listCourses: async () => ({ courses: [], nextPageToken: "" }) },
  ],
  [GradesService, { listEvaluations: async () => ({ evaluations: [] }) }],
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

describe("GradesPage — grades.override gate (T-13)", () => {
  it("S-01a: grades.override → renders SchemeManagementView", async () => {
    renderComponent(<GradesPage />, {
      session: session(["grades.override"]),
      transport: minimalTransport,
    });

    // SchemeManagementView renders this subtitle immediately on mount.
    expect(
      await screen.findByText(
        "Administra los esquemas de evaluación por asignatura.",
      ),
    ).toBeInTheDocument();

    // Placeholder text must NOT appear.
    expect(
      screen.queryByText("Registro de notas — próximamente."),
    ).not.toBeInTheDocument();
  });

  it("S-01b: grades.write (no grades.override) → renders placeholder", () => {
    renderComponent(<GradesPage />, {
      session: session(["grades.write"]),
    });

    expect(
      screen.getByText("Registro de notas — próximamente."),
    ).toBeInTheDocument();

    // SchemeManagementView subtitle must NOT appear.
    expect(
      screen.queryByText(
        "Administra los esquemas de evaluación por asignatura.",
      ),
    ).not.toBeInTheDocument();
  });

  it("S-01c: grades.read (no grades.override) → renders placeholder", () => {
    renderComponent(<GradesPage />, {
      session: session(["grades.read"]),
    });

    expect(
      screen.getByText("Registro de notas — próximamente."),
    ).toBeInTheDocument();
  });

  it("loading session → renders placeholder", () => {
    renderComponent(<GradesPage />, {
      session: { status: "loading" },
    });

    // hasPermission returns false during loading → placeholder, never SchemeManagementView.
    expect(
      screen.getByText("Registro de notas — próximamente."),
    ).toBeInTheDocument();

    // SchemeManagementView subtitle absent → view was never mounted.
    expect(
      screen.queryByText(
        "Administra los esquemas de evaluación por asignatura.",
      ),
    ).not.toBeInTheDocument();
  });

  it("unauthenticated session → renders placeholder", () => {
    renderComponent(<GradesPage />, {
      session: { status: "unauthenticated" },
    });

    expect(
      screen.getByText("Registro de notas — próximamente."),
    ).toBeInTheDocument();

    expect(
      screen.queryByText(
        "Administra los esquemas de evaluación por asignatura.",
      ),
    ).not.toBeInTheDocument();
  });
});
