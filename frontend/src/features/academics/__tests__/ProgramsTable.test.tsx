import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import type { AuthenticatedSession } from "@/features/auth";
import {
  CatalogService,
  ListProgramsResponseSchema,
  ProgramSchema,
} from "@/gen/catalog/v1/catalog_pb";
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

const adminSession = {
  status: "authenticated" as const,
  userId: "1",
  email: "admin@test.com",
  roles: ["admin"],
  permissions: ["catalog.manage"],
};

// A sessionSource that always resolves the admin session — prevents the auth
// guard from redirecting to /login when search navigation triggers ensureQueryData
// after gcTime:0 GC in the test QueryClient.
const adminSessionSource = {
  getSession: async (): Promise<AuthenticatedSession> => ({
    userId: adminSession.userId,
    email: adminSession.email,
    roles: adminSession.roles,
    permissions: adminSession.permissions,
  }),
};

function renderPage(handlers: CatalogImpl) {
  return renderWithProviders({
    route: "/academics",
    transport: makeStubTransport([CatalogService, handlers]),
    session: adminSession,
    sessionSource: adminSessionSource,
  });
}

function renderPageWithRoute(route: string, handlers: CatalogImpl) {
  return renderWithProviders({
    route,
    transport: makeStubTransport([CatalogService, handlers]),
    session: adminSession,
    sessionSource: adminSessionSource,
  });
}

describe("ProgramsTable", () => {
  it("S-01: shows aria-busy skeleton while listPrograms is pending", async () => {
    // Never resolves — keeps the component in the loading state.
    // biome-ignore lint/suspicious/noEmptyBlockStatements: intentional never-resolving promise for testing loading state
    renderPage({ listPrograms: () => new Promise(() => {}) });

    const skeleton = await screen.findByRole("status", {
      name: "Cargando carreras",
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

    await screen.findByText("Todavía no hay carreras");
    const createButtons = screen.getAllByRole("button", {
      name: /crear carrera/i,
    });
    expect(createButtons.length).toBeGreaterThanOrEqual(1);
  });

  it("S-04: transport error shows inline error and retry affordance, no raw codes", async () => {
    renderPage({
      listPrograms: async () => {
        throw new ConnectError("unavailable", Code.Unavailable);
      },
    });

    await screen.findByText(/No se pudo cargar la lista de carreras/);
    expect(
      screen.getByRole("button", { name: /reintentar/i }),
    ).toBeInTheDocument();

    expect(screen.queryByText(/Unavailable/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Code\./)).not.toBeInTheDocument();
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

  // The route guard (/academics requires catalog.manage) prevents reaching this page
  // without the permission. The in-component check is defense-in-depth tested below
  // through the same session that passes the guard — both levels are consistent.
  it("S-16: shows Editar/Eliminar actions when the session has catalog.manage", async () => {
    renderPage({
      listPrograms: async () => ({ programs: [program1] }),
    });

    await screen.findByText("ING-01");
    // Actions button (MoreHorizontal trigger) should be present for manage sessions.
    expect(
      screen.getByRole("button", { name: `Acciones ${program1.code}` }),
    ).toBeInTheDocument();
  });

  it("S-P04: debounce — listPrograms called with typed query after debounce", async () => {
    const user = userEvent.setup();
    const listPrograms = vi.fn(async () =>
      create(ListProgramsResponseSchema, { programs: [], nextPageToken: "" }),
    );

    renderPage({ listPrograms });

    await screen.findByText("Todavía no hay carreras");

    const input = screen.getByPlaceholderText(/buscar/i);
    await user.type(input, "ing");

    await waitFor(
      () => {
        const allCalls = listPrograms.mock.calls as unknown as Array<
          [{ query: string }]
        >;
        const lastCall = allCalls[allCalls.length - 1];
        expect(lastCall?.[0].query).toBe("ing");
      },
      { timeout: 1000 },
    );
  });

  it("S-P05: deep-link ?q=ing applies filter on first render", async () => {
    const listPrograms = vi.fn(async () =>
      create(ListProgramsResponseSchema, { programs: [], nextPageToken: "" }),
    );

    renderPageWithRoute("/academics?q=ing", { listPrograms });

    await waitFor(() => {
      expect(listPrograms).toHaveBeenCalled();
    });

    const calls = listPrograms.mock.calls as unknown as Array<
      [{ query: string }]
    >;
    expect(calls[0][0].query).toBe("ing");

    // No empty-query call before the filter call.
    const emptyQueryCalls = calls.filter((c) => c[0].query === "");
    expect(emptyQueryCalls).toHaveLength(0);
  });

  it("S-P06: Cargar más appends programs, prior rows remain", async () => {
    const user = userEvent.setup();
    let callCount = 0;
    const listPrograms = vi.fn(async () => {
      callCount++;
      if (callCount === 1) {
        return create(ListProgramsResponseSchema, {
          programs: [program1],
          nextPageToken: "cursor-page-2",
        });
      }
      return create(ListProgramsResponseSchema, {
        programs: [program2],
        nextPageToken: "",
      });
    });

    renderPage({ listPrograms });

    await screen.findByText("ING-01");
    expect(
      screen.getByRole("button", { name: /cargar más/i }),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /cargar más/i }));

    await screen.findByText("MED-01");
    expect(screen.getByText("ING-01")).toBeInTheDocument();
  });

  it("S-P07: no Cargar más when nextPageToken is empty", async () => {
    renderPage({
      listPrograms: async () =>
        create(ListProgramsResponseSchema, {
          programs: [program1],
          nextPageToken: "",
        }),
    });

    await screen.findByText("ING-01");
    expect(
      screen.queryByRole("button", { name: /cargar más/i }),
    ).not.toBeInTheDocument();
  });

  it("S-P08: fetchNextPage failure shows toast, existing rows remain", async () => {
    let callCount = 0;
    renderPage({
      listPrograms: vi.fn(async () => {
        callCount++;
        if (callCount === 1) {
          return create(ListProgramsResponseSchema, {
            programs: [program1],
            nextPageToken: "cursor-page-2",
          });
        }
        throw new ConnectError("unavailable", Code.Unavailable);
      }),
    });

    const user = userEvent.setup();
    await screen.findByText("ING-01");

    await user.click(screen.getByRole("button", { name: /cargar más/i }));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith(
        "No se pudieron cargar más carreras.",
      );
    });
    // Existing rows remain visible.
    expect(screen.getByText("ING-01")).toBeInTheDocument();
    expect(
      screen.queryByText("No se pudo cargar la lista de carreras."),
    ).not.toBeInTheDocument();
  });

  it("S-P09: search change resets to page 1 (discards accumulated pages)", async () => {
    const user = userEvent.setup();

    const program3 = create(ProgramSchema, {
      id: "p3",
      code: "ADM-01",
      name: "Administración",
      createdAt: "2024-03-01T00:00:00Z",
      updatedAt: "2024-03-01T00:00:00Z",
    });

    let callCount = 0;
    renderPage({
      listPrograms: vi.fn(async () => {
        callCount++;
        if (callCount === 1) {
          return create(ListProgramsResponseSchema, {
            programs: [program1],
            nextPageToken: "cursor-page-2",
          });
        }
        if (callCount === 2) {
          return create(ListProgramsResponseSchema, {
            programs: [program2],
            nextPageToken: "",
          });
        }
        // New search resets to page 1.
        return create(ListProgramsResponseSchema, {
          programs: [program3],
          nextPageToken: "",
        });
      }),
    });

    await screen.findByText("ING-01");

    // Load page 2.
    await user.click(screen.getByRole("button", { name: /cargar más/i }));
    await screen.findByText("MED-01");
    expect(screen.getByText("ING-01")).toBeInTheDocument();

    // Type a new search term — should reset.
    const input = screen.getByPlaceholderText(/buscar/i);
    await user.clear(input);
    await user.type(input, "adm");

    await screen.findByText("ADM-01");
    expect(screen.queryByText("MED-01")).not.toBeInTheDocument();
    expect(screen.queryByText("ING-01")).not.toBeInTheDocument();
  });

  it("S-P10: pageSize change resets accumulated pages", async () => {
    const user = userEvent.setup();
    let callCount = 0;
    renderPage({
      listPrograms: vi.fn(async () => {
        callCount++;
        if (callCount === 1) {
          return create(ListProgramsResponseSchema, {
            programs: [program1],
            nextPageToken: "cursor-page-2",
          });
        }
        if (callCount === 2) {
          return create(ListProgramsResponseSchema, {
            programs: [program2],
            nextPageToken: "",
          });
        }
        // After pageSize change, page 1 of new size.
        return create(ListProgramsResponseSchema, {
          programs: [program2],
          nextPageToken: "",
        });
      }),
    });

    await screen.findByText("ING-01");

    // Load page 2.
    await user.click(screen.getByRole("button", { name: /cargar más/i }));
    await screen.findByText("MED-01");

    // Change page size — the accumulated pages should reset.
    await user.click(screen.getByRole("combobox"));
    await user.click(await screen.findByText("50 por página"));

    await waitFor(
      () => {
        const allCalls = screen.queryByText("ING-01");
        // After page size change, only page 1 of the new query should be shown.
        // ING-01 is absent because the new call returns only program2.
        expect(allCalls).not.toBeInTheDocument();
      },
      { timeout: 1500 },
    );
  });
});
