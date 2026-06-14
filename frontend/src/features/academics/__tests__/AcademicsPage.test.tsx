import type { ServiceImpl } from "@connectrpc/connect";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import type { AuthenticatedSession } from "@/features/auth";
import { CatalogService } from "@/gen/catalog/v1/catalog_pb";
import { renderWithProviders } from "@/test";

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

type CatalogImpl = Partial<ServiceImpl<typeof CatalogService>>;

const emptyHandlers: CatalogImpl = {
  listPrograms: async () => ({ programs: [] }),
  listCourses: async () => ({ courses: [] }),
};

const adminSession = {
  status: "authenticated" as const,
  userId: "1",
  email: "admin@test.com",
  roles: ["admin"],
  permissions: ["catalog.manage"],
};

// A sessionSource that always returns the admin session — prevents the auth
// guard from redirecting to /login when tab navigation triggers ensureQueryData
// with staleTime:0 in the test QueryClient.
const adminSessionSource = {
  getSession: async (): Promise<AuthenticatedSession> => ({
    userId: adminSession.userId,
    email: adminSession.email,
    roles: adminSession.roles,
    permissions: adminSession.permissions,
  }),
};

function renderAcademicsPage(
  route = "/academics",
  handlers: CatalogImpl = emptyHandlers,
) {
  return renderWithProviders({
    route,
    transport: makeStubTransport([CatalogService, handlers]),
    session: adminSession,
    sessionSource: adminSessionSource,
  });
}

describe("AcademicsPage tab shell", () => {
  it("SC-01: /academics with no ?tab shows programs tab as default and programs content", async () => {
    const { router } = renderAcademicsPage("/academics");

    await screen.findByRole("heading", { name: "Académico" });

    // The "Carreras" tab trigger should be selected by default.
    const programsTab = screen.getByRole("tab", { name: "Carreras" });
    expect(programsTab).toHaveAttribute("aria-selected", "true");

    // At least one "Crear carrera" button is shown (header + possibly empty-state CTA).
    const createCarreraButtons = screen.getAllByRole("button", {
      name: /crear carrera/i,
    });
    expect(createCarreraButtons.length).toBeGreaterThanOrEqual(1);

    // Router search should default to programs (searchStr is either empty or contains programs).
    const searchStr = router.state.location.searchStr;
    expect(
      searchStr === "" ||
        searchStr.includes("programs") ||
        !searchStr.includes("courses"),
    ).toBe(true);
  });

  it("SC-02: clicking 'Asignaturas' tab updates URL search to tab=courses", async () => {
    const user = userEvent.setup();
    const { router } = renderAcademicsPage("/academics");

    await screen.findByRole("heading", { name: "Académico" });

    const coursesTab = screen.getByRole("tab", { name: "Asignaturas" });
    await user.click(coursesTab);

    await waitFor(() => {
      expect(router.state.location.searchStr).toContain("tab=courses");
    });

    expect(coursesTab).toHaveAttribute("aria-selected", "true");
  });

  it("SC-03: clicking 'Carreras' tab from courses updates URL search to tab=programs", async () => {
    const user = userEvent.setup();
    const { router } = renderAcademicsPage("/academics?tab=courses");

    await screen.findByRole("heading", { name: "Académico" });

    const programsTab = screen.getByRole("tab", { name: "Carreras" });
    await user.click(programsTab);

    await waitFor(() => {
      expect(router.state.location.searchStr).toContain("tab=programs");
    });

    expect(programsTab).toHaveAttribute("aria-selected", "true");
  });

  it("SC-05: /academics?tab=bogus falls back to programs (catch in validateSearch)", async () => {
    renderAcademicsPage("/academics?tab=bogus");

    await screen.findByRole("heading", { name: "Académico" });

    // The "Carreras" tab should be active (fallback from bogus value).
    const programsTab = screen.getByRole("tab", { name: "Carreras" });
    expect(programsTab).toHaveAttribute("aria-selected", "true");

    // Programs content is visible: at least one "Crear carrera" button present.
    const createCarreraButtons = screen.getAllByRole("button", {
      name: /crear carrera/i,
    });
    expect(createCarreraButtons.length).toBeGreaterThanOrEqual(1);
  });
});
