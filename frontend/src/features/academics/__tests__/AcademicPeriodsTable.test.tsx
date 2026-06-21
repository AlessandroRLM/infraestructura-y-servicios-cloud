import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import type { AuthenticatedSession } from "@/features/auth";
import {
  AcademicPeriodSchema,
  CatalogService,
} from "@/gen/catalog/v1/catalog_pb";
import { renderWithProviders } from "@/test";

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

const period1 = create(AcademicPeriodSchema, {
  id: "p1",
  year: 2025,
  term: 1,
  startDate: "2025-03-01",
  endDate: "2025-07-15",
  createdAt: "2024-12-01T00:00:00Z",
  updatedAt: "2024-12-01T00:00:00Z",
});

const period2 = create(AcademicPeriodSchema, {
  id: "p2",
  year: 2025,
  term: 2,
  startDate: "2025-08-01",
  endDate: "2025-12-15",
  createdAt: "2024-12-01T00:00:00Z",
  updatedAt: "2024-12-01T00:00:00Z",
});

type CatalogImpl = Partial<ServiceImpl<typeof CatalogService>>;

const adminSession = {
  status: "authenticated" as const,
  userId: "1",
  email: "admin@test.com",
  roles: ["admin"],
  permissions: ["catalog.manage"],
};

const adminSessionSource = {
  getSession: async (): Promise<AuthenticatedSession> => ({
    userId: adminSession.userId,
    email: adminSession.email,
    roles: adminSession.roles,
    permissions: adminSession.permissions,
  }),
};

function renderPeriodsTab(handlers: CatalogImpl) {
  return renderWithProviders({
    route: "/admin/academics?tab=periods",
    transport: makeStubTransport([CatalogService, handlers]),
    session: adminSession,
    sessionSource: adminSessionSource,
  });
}

describe("AcademicPeriodsTable", () => {
  it("shows aria-busy skeleton while listAcademicPeriods is pending", async () => {
    // biome-ignore lint/suspicious/noEmptyBlockStatements: intentional never-resolving promise for testing loading state
    renderPeriodsTab({ listAcademicPeriods: () => new Promise(() => {}) });

    const skeleton = await screen.findByRole("status", {
      name: "Cargando períodos",
    });
    expect(skeleton).toBeInTheDocument();
    expect(skeleton).toHaveAttribute("aria-busy", "true");
  });

  it("shows populated rows with correct columns", async () => {
    renderPeriodsTab({
      listAcademicPeriods: async () => ({
        academicPeriods: [period1, period2],
      }),
    });

    // Wait for table rows to appear via the start date which is unique
    await screen.findByText("2025-03-01");
    expect(screen.getByText("2025-07-15")).toBeInTheDocument();

    expect(
      screen.getByRole("columnheader", { name: "Año" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("columnheader", { name: "Semestre" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("columnheader", { name: "Inicio" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("columnheader", { name: "Término" }),
    ).toBeInTheDocument();
  });

  it("empty state admin shows copy and Crear CTA", async () => {
    renderPeriodsTab({
      listAcademicPeriods: async () => ({ academicPeriods: [] }),
    });

    await screen.findByText("Todavía no hay períodos académicos");
    const createButtons = screen.getAllByRole("button", {
      name: /crear período/i,
    });
    expect(createButtons.length).toBeGreaterThanOrEqual(1);
  });

  it("transport error shows inline error and retry affordance, no raw codes", async () => {
    renderPeriodsTab({
      listAcademicPeriods: async () => {
        throw new ConnectError("unavailable", Code.Unavailable);
      },
    });

    await screen.findByText(/No se pudo cargar la lista de períodos/);
    expect(
      screen.getByRole("button", { name: /reintentar/i }),
    ).toBeInTheDocument();

    expect(screen.queryByText(/Unavailable/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Code\./)).not.toBeInTheDocument();
  });

  it("retry calls listAcademicPeriods again and re-renders rows", async () => {
    const user = userEvent.setup();
    let callCount = 0;
    const listAcademicPeriods = vi.fn(async () => {
      callCount++;
      if (callCount === 1) {
        throw new ConnectError("unavailable", Code.Unavailable);
      }
      return { academicPeriods: [period1] };
    });

    renderPeriodsTab({ listAcademicPeriods });

    await screen.findByRole("button", { name: /reintentar/i });
    await user.click(screen.getByRole("button", { name: /reintentar/i }));

    await screen.findByText("2025-03-01");
  });

  it("shows Editar/Eliminar actions when the session has catalog.manage", async () => {
    renderPeriodsTab({
      listAcademicPeriods: async () => ({ academicPeriods: [period1] }),
    });

    await screen.findByText("2025-03-01");

    expect(
      screen.getByRole("button", {
        name: /editar período 2025 · Semestre 1/i,
      }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", {
        name: /eliminar período 2025 · Semestre 1/i,
      }),
    ).toBeInTheDocument();
  });
});
