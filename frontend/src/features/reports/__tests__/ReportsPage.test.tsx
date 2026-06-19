/**
 * ReportsPage tab shell tests — AC-1.a, AC-1.b, AC-1.c, AC-1.d, AC-2.e
 *
 * Permission-gate tests use ReportsTabShell directly (no router dependency).
 * URL navigation tests use renderWithProviders with the full route.
 */
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { AuthenticatedSession } from "@/features/auth";
import { renderComponent, renderWithProviders } from "@/test";
import { ReportsTabShell } from "../components/ReportsPage";

// biome-ignore lint/suspicious/noEmptyBlockStatements: intentional no-op for tests
const noOp = () => {};

function makeSession(permissions: string[]) {
  return {
    status: "authenticated" as const,
    userId: "u-test",
    email: "test@test.com",
    roles: ["admin"],
    permissions,
  };
}

function makeSessionSource(session: AuthenticatedSession | null) {
  return { getSession: async () => session };
}

const teacherSession = makeSession(["reports.read"]);
const adminSession = makeSession(["reports.read", "users.manage"]);
const noPermSession = makeSession([]);

const teacherSource = makeSessionSource({
  userId: teacherSession.userId,
  email: teacherSession.email,
  roles: teacherSession.roles,
  permissions: teacherSession.permissions,
});

const adminSource = makeSessionSource({
  userId: adminSession.userId,
  email: adminSession.email,
  roles: adminSession.roles,
  permissions: adminSession.permissions,
});

describe("ReportsTabShell — permission gate (isolated, no router dependency)", () => {
  it("AC-1.a: reports.read only → exactly 1 tab 'Calificaciones por Sección'", () => {
    renderComponent(
      <ReportsTabShell activeTab="section-grade" onTabChange={noOp} />,
      { session: teacherSession },
    );

    const tabs = screen.getAllByRole("tab");
    expect(tabs).toHaveLength(1);
    expect(tabs[0]).toHaveAccessibleName("Calificaciones por Sección");
  });

  it("AC-1.b: reports.read + users.manage → exactly 4 tabs", () => {
    renderComponent(
      <ReportsTabShell activeTab="section-grade" onTabChange={noOp} />,
      { session: adminSession },
    );

    const tabs = screen.getAllByRole("tab");
    expect(tabs).toHaveLength(4);

    const tabNames = tabs.map((t) => t.textContent);
    expect(tabNames).toContain("Calificaciones por Sección");
    expect(tabNames).toContain("Ocupación por Período");
    expect(tabNames).toContain("Resumen de Programa");
    expect(tabNames).toContain("Expediente de Alumno");
  });

  it("AC-1.c: no reports.read → 0 tabs, permission-empty state", () => {
    renderComponent(
      <ReportsTabShell activeTab="section-grade" onTabChange={noOp} />,
      { session: noPermSession },
    );

    expect(screen.queryAllByRole("tab")).toHaveLength(0);
    expect(
      screen.getByText("No tienes permisos para acceder a los reportes."),
    ).toBeInTheDocument();
  });

  it("AC-1.d: no tab has a 'disabled' attribute", () => {
    renderComponent(
      <ReportsTabShell activeTab="section-grade" onTabChange={noOp} />,
      { session: adminSession },
    );

    const tabs = screen.getAllByRole("tab");
    for (const tab of tabs) {
      expect(tab).not.toHaveAttribute("disabled");
    }
  });

  it("StudentRecord tab absent without users.manage (RF-1.3)", () => {
    renderComponent(
      <ReportsTabShell activeTab="section-grade" onTabChange={noOp} />,
      { session: teacherSession },
    );

    expect(
      screen.queryByRole("tab", { name: "Expediente de Alumno" }),
    ).not.toBeInTheDocument();
  });

  it("onTabChange called when tab is clicked", async () => {
    const user = userEvent.setup();
    const onTabChange = vi.fn();

    renderComponent(
      <ReportsTabShell activeTab="section-grade" onTabChange={onTabChange} />,
      { session: adminSession },
    );

    await user.click(
      screen.getByRole("tab", { name: "Ocupación por Período" }),
    );
    expect(onTabChange).toHaveBeenCalledWith("occupancy");
  });
});

describe("ReportsPage — URL param navigation (with router)", () => {
  it("AC-2.e: clicking 'Ocupación por Período' updates URL param tab=occupancy", async () => {
    const user = userEvent.setup();
    const { router } = renderWithProviders({
      route: "/admin/reports",
      session: adminSession,
      sessionSource: adminSource,
    });

    await screen.findByRole("heading", { name: "Reportes" });

    const occupancyTab = screen.getByRole("tab", {
      name: "Ocupación por Período",
    });
    await user.click(occupancyTab);

    await waitFor(() => {
      expect(router.state.location.searchStr).toContain("tab=occupancy");
    });
  });

  it("teacher session → 1 tab in full router context", async () => {
    renderWithProviders({
      route: "/admin/reports",
      session: teacherSession,
      sessionSource: teacherSource,
    });

    await screen.findByRole("heading", { name: "Reportes" });

    const tabs = screen.getAllByRole("tab");
    expect(tabs).toHaveLength(1);
  });

  /**
   * RF-2.2 / Round-1 fix #6: permission-fallback URL correction.
   * A TEACHER (reports.read only, no users.manage) who lands on
   * ?tab=student-record must have the URL corrected to tab=section-grade,
   * and the visible panel must be the SectionGrade tab.
   */
  it("RF-2.2: teacher landing on ?tab=student-record gets URL corrected to tab=section-grade", async () => {
    const { router } = renderWithProviders({
      route: "/admin/reports?tab=student-record",
      session: teacherSession,
      sessionSource: teacherSource,
    });

    await screen.findByRole("heading", { name: "Reportes" });

    // URL must no longer show student-record.
    await waitFor(() => {
      expect(router.state.location.searchStr).not.toContain("student-record");
      expect(router.state.location.searchStr).toContain("tab=section-grade");
    });

    // The only visible tab must be "Calificaciones por Sección" (section-grade).
    const tabs = screen.getAllByRole("tab");
    expect(tabs).toHaveLength(1);
    expect(tabs[0]).toHaveAccessibleName("Calificaciones por Sección");
  });
});
