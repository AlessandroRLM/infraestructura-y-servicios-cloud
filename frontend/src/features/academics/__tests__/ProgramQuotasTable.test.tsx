/**
 * ProgramQuotasTable — rendered through the per-program Sheet.
 * The "Cupos" tab inside the Sheet owns the programId context; the table
 * itself never receives an empty programId.
 */
import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import type { AuthenticatedSession } from "@/features/auth";
import {
  CatalogService,
  ProgramQuotaSchema,
  ProgramSchema,
} from "@/gen/catalog/v1/catalog_pb";
import { renderWithProviders } from "@/test";

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

const mockProgram = create(ProgramSchema, {
  id: "p1",
  code: "ING",
  name: "Ingeniería Civil",
  createdAt: "2024-01-01",
  updatedAt: "2024-01-01",
});

const quota1 = create(ProgramQuotaSchema, {
  id: "q1",
  programId: "p1",
  year: 2025,
  admissionQuota: 60,
  createdAt: "2024-01-01",
  updatedAt: "2024-01-01",
});

const quota2 = create(ProgramQuotaSchema, {
  id: "q2",
  programId: "p1",
  year: 2024,
  admissionQuota: 40,
  createdAt: "2024-01-01",
  updatedAt: "2024-01-01",
});

type CatalogImpl = Partial<ServiceImpl<typeof CatalogService>>;

const adminSession: AuthenticatedSession = {
  userId: "1",
  email: "admin@test.com",
  roles: ["admin"],
  permissions: ["catalog.manage"],
};

const adminSessionSource = {
  getSession: async (): Promise<AuthenticatedSession> => adminSession,
};

async function renderViaSheet(handlers: CatalogImpl) {
  const user = userEvent.setup();
  renderWithProviders({
    route: "/admin/academics",
    transport: makeStubTransport([CatalogService, handlers]),
    session: { status: "authenticated", ...adminSession },
    sessionSource: adminSessionSource,
  });
  await screen.findByText("ING");
  await user.click(screen.getByRole("button", { name: "Acciones ING" }));
  await user.click(screen.getByRole("menuitem", { name: /gestionar/i }));
  await user.click(screen.getByRole("tab", { name: "Cupos" }));
  return { user };
}

describe("ProgramQuotasTable", () => {
  it("shows aria-busy skeleton while listProgramQuotas is pending", async () => {
    await renderViaSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      // biome-ignore lint/suspicious/noEmptyBlockStatements: intentional never-resolving promise for testing loading state
      listProgramQuotas: () => new Promise(() => {}),
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    const skeleton = await screen.findByRole("status", {
      name: "Cargando cupos",
    });
    expect(skeleton).toBeInTheDocument();
    expect(skeleton).toHaveAttribute("aria-busy", "true");
  });

  it("shows populated rows with correct Año and Cupo columns (no Carrera column)", async () => {
    await renderViaSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramQuotas: async () => ({ programQuotas: [quota1, quota2] }),
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    await screen.findByText("60");
    expect(screen.getByText("40")).toBeInTheDocument();

    expect(
      screen.queryByRole("columnheader", { name: "Carrera" }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole("columnheader", { name: "Año" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("columnheader", { name: "Cupo" }),
    ).toBeInTheDocument();
  });

  it("renders year and admissionQuota values in cells", async () => {
    await renderViaSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramQuotas: async () => ({ programQuotas: [quota1] }),
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    await screen.findByText("60");
    expect(screen.getByText("2025")).toBeInTheDocument();
  });

  it("empty state shows copy and Crear CTA", async () => {
    await renderViaSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramQuotas: async () => ({ programQuotas: [] }),
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    await screen.findByText("Todavía no hay cupos");
    const createButtons = screen.getAllByRole("button", {
      name: /crear cupo/i,
    });
    expect(createButtons.length).toBeGreaterThanOrEqual(1);
  });

  it("transport error shows inline error and retry affordance, no raw codes", async () => {
    await renderViaSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramQuotas: async () => {
        throw new ConnectError("unavailable", Code.Unavailable);
      },
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    await screen.findByText(/No se pudo cargar la lista de cupos/);
    expect(
      screen.getByRole("button", { name: /reintentar/i }),
    ).toBeInTheDocument();

    expect(screen.queryByText(/Unavailable/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Code\./)).not.toBeInTheDocument();
  });

  it("retry calls listProgramQuotas again and re-renders rows", async () => {
    const user = userEvent.setup();
    let callCount = 0;
    const listProgramQuotas = vi.fn(async () => {
      callCount++;
      if (callCount === 1) {
        throw new ConnectError("unavailable", Code.Unavailable);
      }
      return { programQuotas: [quota1] };
    });

    renderWithProviders({
      route: "/admin/academics",
      transport: makeStubTransport([
        CatalogService,
        {
          listPrograms: async () => ({ programs: [mockProgram] }),
          listProgramQuotas,
          listProgramCourses: async () => ({ programCourses: [] }),
          listCourses: async () => ({ courses: [] }),
        },
      ]),
      session: { status: "authenticated", ...adminSession },
      sessionSource: adminSessionSource,
    });

    await screen.findByText("ING");
    await user.click(screen.getByRole("button", { name: "Acciones ING" }));
    await user.click(screen.getByRole("menuitem", { name: /gestionar/i }));
    await user.click(screen.getByRole("tab", { name: "Cupos" }));

    await screen.findByRole("button", { name: /reintentar/i });
    await user.click(screen.getByRole("button", { name: /reintentar/i }));

    await screen.findByText("60");
  });

  it("shows Editar/Eliminar actions when the session has catalog.manage", async () => {
    await renderViaSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramQuotas: async () => ({ programQuotas: [quota1] }),
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    await screen.findByText("60");
    expect(
      screen.getByRole("button", { name: `Editar cupo ${quota1.id}` }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: `Eliminar cupo ${quota1.id}` }),
    ).toBeInTheDocument();
  });

  it("quota list has no Cargar más button (flat list, not paginated)", async () => {
    await renderViaSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramQuotas: async () => ({ programQuotas: [quota1, quota2] }),
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    await screen.findByText("60");
    expect(
      screen.queryByRole("button", { name: /cargar más/i }),
    ).not.toBeInTheDocument();
  });
});
