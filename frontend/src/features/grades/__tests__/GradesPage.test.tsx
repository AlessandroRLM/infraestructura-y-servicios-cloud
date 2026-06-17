import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { makeStubTransport } from "@/core/test";
import type { SessionState } from "@/features/auth";
import { CatalogService } from "@/gen/catalog/v1/catalog_pb";
import { renderComponent } from "@/test";
import { GradesPage } from "../components/GradesPage";

/** Minimal catalog stub to satisfy useCourses inside SchemeManagementView. */
const minimalCatalogTransport = makeStubTransport([
  CatalogService,
  { listCourses: async () => ({ courses: [], nextPageToken: "" }) },
]);

function session(permissions: string[]): SessionState {
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
      transport: minimalCatalogTransport,
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

  it("loading session → renders placeholder (no server call while unresolved)", () => {
    renderComponent(<GradesPage />, {
      session: { status: "loading" },
    });

    // hasPermission returns false during loading → placeholder, never SchemeManagementView.
    expect(
      screen.getByText("Registro de notas — próximamente."),
    ).toBeInTheDocument();
  });
});
