import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import { CatalogService, CourseSchema } from "@/gen/catalog/v1/catalog_pb";
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

function renderCoursesTab(
  handlers: CatalogImpl,
  permissions: string[] = ["catalog.manage"],
) {
  return renderWithProviders({
    route: "/academics?tab=courses",
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

  it("SC-09: empty state non-admin shows copy but no Crear CTA", async () => {
    renderCoursesTab({ listCourses: async () => ({ courses: [] }) }, []);

    await screen.findByText("Todavía no hay asignaturas");
    expect(
      screen.queryByRole("button", { name: /crear asignatura/i }),
    ).not.toBeInTheDocument();
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

  it("SC-12: non-admin sees rows but no Editar/Eliminar/Crear buttons", async () => {
    renderCoursesTab({ listCourses: async () => ({ courses: [course1] }) }, []);

    await screen.findByText("CS-101");
    expect(
      screen.queryByRole("button", { name: /editar/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /eliminar/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /crear asignatura/i }),
    ).not.toBeInTheDocument();
  });
});
