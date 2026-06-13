import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import { CatalogService, ProgramSchema } from "@/gen/catalog/v1/catalog_pb";
import { renderWithProviders } from "@/test";

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

const program1 = create(ProgramSchema, {
  id: "p1",
  code: "ING-01",
  name: "Ingeniería de Software",
  createdAt: "2024-01-15T00:00:00Z",
  updatedAt: "2024-01-15T00:00:00Z",
});

const program2 = create(ProgramSchema, {
  id: "p2",
  code: "MED-01",
  name: "Medicina",
  createdAt: "2024-02-01T00:00:00Z",
  updatedAt: "2024-02-01T00:00:00Z",
});

type CatalogImpl = Partial<ServiceImpl<typeof CatalogService>>;

function renderPage(
  handlers: CatalogImpl,
  permissions: string[] = ["catalog.manage"],
) {
  return renderWithProviders({
    route: "/academics",
    transport: makeStubTransport([CatalogService, handlers]),
    session: {
      status: "authenticated",
      userId: "1",
      email: "admin@test.com",
      roles: ["admin"],
      permissions,
    },
  });
}

describe("ProgramsTable", () => {
  it("S-01: shows aria-busy skeleton while listPrograms is pending", async () => {
    // Never resolves — keeps the component in the loading state.
    // biome-ignore lint/suspicious/noEmptyBlockStatements: intentional never-resolving promise for testing loading state
    renderPage({ listPrograms: () => new Promise(() => {}) });

    const skeleton = await screen.findByRole("status", {
      name: "Cargando programas",
    });
    expect(skeleton).toBeInTheDocument();
    expect(skeleton).toHaveAttribute("aria-busy", "true");
  });

  it("S-02: shows populated rows with correct columns", async () => {
    renderPage({
      listPrograms: async () => ({ programs: [program1, program2] }),
    });

    await screen.findByText("ING-01");
    expect(screen.getByText("Ingeniería de Software")).toBeInTheDocument();
    expect(screen.getByText("MED-01")).toBeInTheDocument();
    expect(screen.getByText("Medicina")).toBeInTheDocument();

    expect(
      screen.getByRole("columnheader", { name: "Código" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("columnheader", { name: "Nombre" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("columnheader", { name: "Creado" }),
    ).toBeInTheDocument();
  });

  it("S-03: shows empty state copy and Crear CTA", async () => {
    renderPage({ listPrograms: async () => ({ programs: [] }) });

    await screen.findByText("Todavía no hay programas");
    const createButtons = screen.getAllByRole("button", {
      name: /crear programa/i,
    });
    expect(createButtons.length).toBeGreaterThanOrEqual(1);
  });

  it("S-04: transport error shows inline error and retry affordance, no raw codes", async () => {
    renderPage({
      listPrograms: async () => {
        throw new ConnectError("unavailable", Code.Unavailable);
      },
    });

    await screen.findByText(/No se pudo cargar la lista de programas/);
    expect(
      screen.getByRole("button", { name: /reintentar/i }),
    ).toBeInTheDocument();

    expect(screen.queryByText(/Unavailable/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Code\./)).not.toBeInTheDocument();
  });

  it("S-16: user without catalog.manage sees no Editar/Eliminar actions and no Crear header button", async () => {
    renderPage({ listPrograms: async () => ({ programs: [program1] }) }, []);

    await screen.findByText("ING-01");
    expect(
      screen.queryByRole("button", { name: /editar/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /eliminar/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /crear programa/i }),
    ).not.toBeInTheDocument();
  });

  it("retry calls listPrograms again", async () => {
    const user = userEvent.setup();
    let callCount = 0;
    const listPrograms = vi.fn(async () => {
      callCount++;
      if (callCount === 1) {
        throw new ConnectError("unavailable", Code.Unavailable);
      }
      return { programs: [program1] };
    });

    renderPage({ listPrograms });

    await screen.findByRole("button", { name: /reintentar/i });
    await user.click(screen.getByRole("button", { name: /reintentar/i }));

    await screen.findByText("ING-01");
  });
});
