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
  CourseSchema,
  ListCoursesResponseSchema,
} from "@/gen/catalog/v1/catalog_pb";
import { renderWithProviders } from "@/test";

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

const course1 = create(CourseSchema, {
  id: "c1",
  code: "CS-101",
  name: "Cálculo",
  credits: 5,
  createdAt: "2024-01-15T00:00:00Z",
  updatedAt: "2024-01-15T00:00:00Z",
});

const course2 = create(CourseSchema, {
  id: "c2",
  code: "FIS-01",
  name: "Física",
  credits: 6,
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

function renderCoursesTab(handlers: CatalogImpl) {
  return renderWithProviders({
    route: "/academics?tab=courses",
    transport: makeStubTransport([CatalogService, handlers]),
    session: adminSession,
    sessionSource: adminSessionSource,
  });
}

function renderCoursesWithRoute(route: string, handlers: CatalogImpl) {
  return renderWithProviders({
    route,
    transport: makeStubTransport([CatalogService, handlers]),
    session: adminSession,
    sessionSource: adminSessionSource,
  });
}

describe("CoursesTable", () => {
  it("SC-06: shows aria-busy skeleton while listCourses is pending", async () => {
    // biome-ignore lint/suspicious/noEmptyBlockStatements: intentional never-resolving promise for testing loading state
    renderCoursesTab({ listCourses: () => new Promise(() => {}) });

    const skeleton = await screen.findByRole("status", {
      name: "Cargando asignaturas",
    });
    expect(skeleton).toBeInTheDocument();
    expect(skeleton).toHaveAttribute("aria-busy", "true");
  });

  it("SC-07: shows populated rows with correct columns", async () => {
    renderCoursesTab({
      listCourses: async () => ({ courses: [course1, course2] }),
    });

    await screen.findByText("CS-101");
    expect(screen.getByText("Cálculo")).toBeInTheDocument();
    expect(screen.getByText("FIS-01")).toBeInTheDocument();
    expect(screen.getByText("Física")).toBeInTheDocument();

    expect(
      screen.getByRole("columnheader", { name: "Código" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("columnheader", { name: "Nombre" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("columnheader", { name: "Créditos" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("columnheader", { name: "Creado" }),
    ).toBeInTheDocument();
  });

  it("SC-08: empty state admin shows copy and Crear CTA", async () => {
    renderCoursesTab({ listCourses: async () => ({ courses: [] }) });

    await screen.findByText("Todavía no hay asignaturas");
    const createButtons = screen.getAllByRole("button", {
      name: /crear asignatura/i,
    });
    expect(createButtons.length).toBeGreaterThanOrEqual(1);
  });

  it("SC-10: transport error shows inline error and retry affordance, no raw codes", async () => {
    renderCoursesTab({
      listCourses: async () => {
        throw new ConnectError("unavailable", Code.Unavailable);
      },
    });

    await screen.findByText(/No se pudo cargar la lista de asignaturas/);
    expect(
      screen.getByRole("button", { name: /reintentar/i }),
    ).toBeInTheDocument();

    expect(screen.queryByText(/Unavailable/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Code\./)).not.toBeInTheDocument();
  });

  it("SC-11: retry calls listCourses again and re-renders rows", async () => {
    const user = userEvent.setup();
    let callCount = 0;
    const listCourses = vi.fn(async () => {
      callCount++;
      if (callCount === 1) {
        throw new ConnectError("unavailable", Code.Unavailable);
      }
      return { courses: [course1] };
    });

    renderCoursesTab({ listCourses });

    await screen.findByRole("button", { name: /reintentar/i });
    await user.click(screen.getByRole("button", { name: /reintentar/i }));

    await screen.findByText("CS-101");
  });

  // The route guard (/academics requires catalog.manage) prevents reaching this page
  // without the permission. The in-component check is defense-in-depth tested below
  // through the same session that passes the guard — both levels are consistent.
  it("SC-12: shows Editar/Eliminar actions when the session has catalog.manage", async () => {
    renderCoursesTab({
      listCourses: async () => ({ courses: [course1] }),
    });

    await screen.findByText("CS-101");
    expect(
      screen.getByRole("button", { name: `Editar ${course1.code}` }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: `Eliminar ${course1.code}` }),
    ).toBeInTheDocument();
  });

  it("SC-P04: debounce — listCourses called with typed query after debounce", async () => {
    const user = userEvent.setup();
    const listCourses = vi.fn(async () =>
      create(ListCoursesResponseSchema, { courses: [], nextPageToken: "" }),
    );

    renderCoursesTab({ listCourses });

    await screen.findByText("Todavía no hay asignaturas");

    const input = screen.getByPlaceholderText(/buscar/i);
    await user.type(input, "cal");

    await waitFor(
      () => {
        const allCalls = listCourses.mock.calls as unknown as Array<
          [{ query: string }]
        >;
        const lastCall = allCalls[allCalls.length - 1];
        expect(lastCall?.[0].query).toBe("cal");
      },
      { timeout: 1000 },
    );
  });

  it("SC-P05: deep-link ?q=cal applies filter on first render", async () => {
    const listCourses = vi.fn(async () =>
      create(ListCoursesResponseSchema, { courses: [], nextPageToken: "" }),
    );

    renderCoursesWithRoute("/academics?tab=courses&q=cal", { listCourses });

    await waitFor(() => {
      expect(listCourses).toHaveBeenCalled();
    });

    const calls = listCourses.mock.calls as unknown as Array<
      [{ query: string }]
    >;
    expect(calls[0][0].query).toBe("cal");

    const emptyQueryCalls = calls.filter((c) => c[0].query === "");
    expect(emptyQueryCalls).toHaveLength(0);
  });

  it("SC-P06: Cargar más appends courses, prior rows remain", async () => {
    const user = userEvent.setup();
    let callCount = 0;
    const listCourses = vi.fn(async () => {
      callCount++;
      if (callCount === 1) {
        return create(ListCoursesResponseSchema, {
          courses: [course1],
          nextPageToken: "cursor-page-2",
        });
      }
      return create(ListCoursesResponseSchema, {
        courses: [course2],
        nextPageToken: "",
      });
    });

    renderCoursesTab({ listCourses });

    await screen.findByText("CS-101");
    expect(
      screen.getByRole("button", { name: /cargar más/i }),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /cargar más/i }));

    await screen.findByText("FIS-01");
    expect(screen.getByText("CS-101")).toBeInTheDocument();
  });

  it("SC-P07: no Cargar más when nextPageToken is empty", async () => {
    renderCoursesTab({
      listCourses: async () =>
        create(ListCoursesResponseSchema, {
          courses: [course1],
          nextPageToken: "",
        }),
    });

    await screen.findByText("CS-101");
    expect(
      screen.queryByRole("button", { name: /cargar más/i }),
    ).not.toBeInTheDocument();
  });

  it("SC-P08: fetchNextPage failure shows toast, existing rows remain", async () => {
    let callCount = 0;
    renderCoursesTab({
      listCourses: vi.fn(async () => {
        callCount++;
        if (callCount === 1) {
          return create(ListCoursesResponseSchema, {
            courses: [course1],
            nextPageToken: "cursor-page-2",
          });
        }
        throw new ConnectError("unavailable", Code.Unavailable);
      }),
    });

    const user = userEvent.setup();
    await screen.findByText("CS-101");

    await user.click(screen.getByRole("button", { name: /cargar más/i }));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith(
        "No se pudieron cargar más asignaturas.",
      );
    });
    expect(screen.getByText("CS-101")).toBeInTheDocument();
    expect(
      screen.queryByText("No se pudo cargar la lista de asignaturas."),
    ).not.toBeInTheDocument();
  });
});
