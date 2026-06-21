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
  ListSectionsResponseSchema,
  SectionSchema,
} from "@/gen/catalog/v1/catalog_pb";
import { renderWithProviders } from "@/test";

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

const section1 = create(SectionSchema, {
  id: "s1",
  courseId: "c1",
  academicPeriodId: "ap1",
  seatCapacity: 30,
  createdAt: "2024-01-15T00:00:00Z",
  updatedAt: "2024-01-15T00:00:00Z",
});

const section2 = create(SectionSchema, {
  id: "s2",
  courseId: "c2",
  academicPeriodId: "ap1",
  seatCapacity: 45,
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

const adminSessionSource = {
  getSession: async (): Promise<AuthenticatedSession> => ({
    userId: adminSession.userId,
    email: adminSession.email,
    roles: adminSession.roles,
    permissions: adminSession.permissions,
  }),
};

function renderSectionsTab(handlers: CatalogImpl) {
  return renderWithProviders({
    route: "/admin/academics?tab=sections",
    transport: makeStubTransport([CatalogService, handlers]),
    session: adminSession,
    sessionSource: adminSessionSource,
  });
}

describe("SectionsTable", () => {
  it("shows aria-busy skeleton while listSections is pending", async () => {
    // biome-ignore lint/suspicious/noEmptyBlockStatements: intentional never-resolving promise for testing loading state
    renderSectionsTab({ listSections: () => new Promise(() => {}) });

    const skeleton = await screen.findByRole("status", {
      name: "Cargando secciones",
    });
    expect(skeleton).toBeInTheDocument();
    expect(skeleton).toHaveAttribute("aria-busy", "true");
  });

  it("shows populated rows with correct columns", async () => {
    renderSectionsTab({
      listSections: async () => ({ sections: [section1, section2] }),
    });

    await screen.findByText("30");
    expect(screen.getByText("45")).toBeInTheDocument();

    expect(
      screen.getByRole("columnheader", { name: "Asignatura" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("columnheader", { name: "Período" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("columnheader", { name: "Capacidad" }),
    ).toBeInTheDocument();
  });

  it("empty state admin shows copy and Crear CTA", async () => {
    renderSectionsTab({ listSections: async () => ({ sections: [] }) });

    await screen.findByText("Todavía no hay secciones");
    const createButtons = screen.getAllByRole("button", {
      name: /crear sección/i,
    });
    expect(createButtons.length).toBeGreaterThanOrEqual(1);
  });

  it("transport error shows inline error and retry affordance, no raw codes", async () => {
    renderSectionsTab({
      listSections: async () => {
        throw new ConnectError("unavailable", Code.Unavailable);
      },
    });

    await screen.findByText(/No se pudo cargar la lista de secciones/);
    expect(
      screen.getByRole("button", { name: /reintentar/i }),
    ).toBeInTheDocument();

    expect(screen.queryByText(/Unavailable/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Code\./)).not.toBeInTheDocument();
  });

  it("retry calls listSections again and re-renders rows", async () => {
    const user = userEvent.setup();
    let callCount = 0;
    const listSections = vi.fn(async () => {
      callCount++;
      if (callCount === 1) {
        throw new ConnectError("unavailable", Code.Unavailable);
      }
      return { sections: [section1] };
    });

    renderSectionsTab({ listSections });

    await screen.findByRole("button", { name: /reintentar/i });
    await user.click(screen.getByRole("button", { name: /reintentar/i }));

    await screen.findByText("30");
  });

  it("shows Editar/Eliminar actions when the session has catalog.manage", async () => {
    renderSectionsTab({
      listSections: async () => ({ sections: [section1] }),
    });

    await screen.findByText("30");
    expect(
      screen.getByRole("button", { name: `Editar sección ${section1.id}` }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: `Eliminar sección ${section1.id}` }),
    ).toBeInTheDocument();
  });

  it("Cargar más appends sections, prior rows remain", async () => {
    const user = userEvent.setup();
    let callCount = 0;
    const listSections = vi.fn(async () => {
      callCount++;
      if (callCount === 1) {
        return create(ListSectionsResponseSchema, {
          sections: [section1],
          nextPageToken: "cursor-page-2",
        });
      }
      return create(ListSectionsResponseSchema, {
        sections: [section2],
        nextPageToken: "",
      });
    });

    renderSectionsTab({ listSections });

    await screen.findByText("30");
    expect(
      screen.getByRole("button", { name: /cargar más/i }),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /cargar más/i }));

    await screen.findByText("45");
    expect(screen.getByText("30")).toBeInTheDocument();
  });

  it("no Cargar más when nextPageToken is empty", async () => {
    renderSectionsTab({
      listSections: async () =>
        create(ListSectionsResponseSchema, {
          sections: [section1],
          nextPageToken: "",
        }),
    });

    await screen.findByText("30");
    expect(
      screen.queryByRole("button", { name: /cargar más/i }),
    ).not.toBeInTheDocument();
  });

  it("fetchNextPage failure shows toast, existing rows remain", async () => {
    let callCount = 0;
    renderSectionsTab({
      listSections: vi.fn(async () => {
        callCount++;
        if (callCount === 1) {
          return create(ListSectionsResponseSchema, {
            sections: [section1],
            nextPageToken: "cursor-page-2",
          });
        }
        throw new ConnectError("unavailable", Code.Unavailable);
      }),
    });

    const user = userEvent.setup();
    await screen.findByText("30");

    await user.click(screen.getByRole("button", { name: /cargar más/i }));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith(
        "No se pudieron cargar más secciones.",
      );
    });
    expect(screen.getByText("30")).toBeInTheDocument();
    expect(
      screen.queryByText("No se pudo cargar la lista de secciones."),
    ).not.toBeInTheDocument();
  });
});
