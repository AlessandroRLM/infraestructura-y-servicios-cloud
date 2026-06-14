import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import {
  CatalogService,
  CourseSchema,
  ProgramCourseSchema,
  ProgramSchema,
  RemoveCourseFromProgramResponseSchema,
} from "@/gen/catalog/v1/catalog_pb";
import { renderWithProviders } from "@/test";

const { toastSuccess, toastError } = vi.hoisted(() => ({
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));
vi.mock("sonner", () => ({
  toast: { success: toastSuccess, error: toastError },
}));

type CatalogImpl = Partial<ServiceImpl<typeof CatalogService>>;

const mockProgram = create(ProgramSchema, {
  id: "prog-1",
  code: "ING-01",
  name: "Ingeniería de Software",
  createdAt: "2024-01-01",
  updatedAt: "2024-01-01",
});

const mat101 = create(CourseSchema, {
  id: "course-mat101",
  code: "MAT101",
  name: "Cálculo I",
  credits: 4,
});

const fis201 = create(CourseSchema, {
  id: "course-fis201",
  code: "FIS201",
  name: "Física I",
  credits: 3,
});

const qui301 = create(CourseSchema, {
  id: "course-qui301",
  code: "QUI301",
  name: "Química Básica",
  credits: 3,
});

const pcMat101 = create(ProgramCourseSchema, {
  programId: "prog-1",
  courseId: "course-mat101",
  course: mat101,
});

async function renderWithSheet(handlers: CatalogImpl) {
  const user = userEvent.setup();
  renderWithProviders({
    route: "/academics",
    transport: makeStubTransport([CatalogService, handlers]),
    session: {
      status: "authenticated",
      userId: "1",
      email: "admin@test.com",
      roles: ["admin"],
      permissions: ["catalog.manage"],
    },
  });
  await screen.findByText("ING-01");
  // Open the ⋯ menu and click Asignaturas
  await user.click(screen.getByRole("button", { name: "Acciones ING-01" }));
  await user.click(screen.getByRole("menuitem", { name: /asignaturas/i }));
  return { user };
}

describe("ProgramCoursesSheet — open from ProgramsTable", () => {
  it("Scenario 5: opens Sheet with correct title via Asignaturas menu item", async () => {
    await renderWithSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    await screen.findByText("Asignaturas de Ingeniería de Software");
  });
});

describe("ProgramCoursesManager — loading state", () => {
  it("Scenario 6: shows loading skeleton while listProgramCourses is pending", async () => {
    await renderWithSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      // biome-ignore lint/suspicious/noEmptyBlockStatements: intentional never-resolving promise for testing loading state
      listProgramCourses: () => new Promise(() => {}),
      listCourses: async () => ({ courses: [] }),
    });

    const skeleton = await screen.findByRole("status", {
      name: "Cargando asignaturas",
    });
    expect(skeleton).toBeInTheDocument();
    expect(skeleton).toHaveAttribute("aria-busy", "true");
  });
});

describe("ProgramCoursesManager — empty state", () => {
  it("Scenario 7: shows empty copy when program has no associated courses", async () => {
    await renderWithSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    await screen.findByText("Esta carrera no tiene asignaturas todavía.");
  });
});

describe("ProgramCoursesManager — associated course list", () => {
  it("Scenario 8: renders code, name, and credits Badge for each associated course", async () => {
    await renderWithSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramCourses: async () => ({ programCourses: [pcMat101] }),
      listCourses: async () => ({ courses: [mat101] }),
    });

    await screen.findByText("MAT101");
    expect(screen.getByText("Cálculo I")).toBeInTheDocument();
    expect(screen.getByText("4 créditos")).toBeInTheDocument();
    expect(
      screen.getByRole("button", {
        name: "Quitar MAT101 de la carrera",
      }),
    ).toBeInTheDocument();
  });
});

describe("ProgramCoursesManager — section error + retry", () => {
  it("Scenario 9: shows inline error and Reintentar button on listProgramCourses error", async () => {
    const user = userEvent.setup();
    let callCount = 0;
    const listProgramCourses = vi.fn(async () => {
      callCount++;
      if (callCount === 1) {
        throw new ConnectError("unavailable", Code.Unavailable);
      }
      return { programCourses: [] };
    });

    renderWithProviders({
      route: "/academics",
      transport: makeStubTransport([
        CatalogService,
        {
          listPrograms: async () => ({ programs: [mockProgram] }),
          listProgramCourses,
          listCourses: async () => ({ courses: [] }),
        },
      ]),
      session: {
        status: "authenticated",
        userId: "1",
        email: "admin@test.com",
        roles: ["admin"],
        permissions: ["catalog.manage"],
      },
    });

    await screen.findByText("ING-01");
    await user.click(screen.getByRole("button", { name: "Acciones ING-01" }));
    await user.click(screen.getByRole("menuitem", { name: /asignaturas/i }));

    await screen.findByText("No se pudo cargar las asignaturas de la carrera.");
    expect(
      screen.getByRole("button", { name: /reintentar/i }),
    ).toBeInTheDocument();

    // Sheet header still visible
    expect(
      screen.getByText("Asignaturas de Ingeniería de Software"),
    ).toBeInTheDocument();

    // Retry triggers refetch
    await user.click(screen.getByRole("button", { name: /reintentar/i }));
    await screen.findByText("Esta carrera no tiene asignaturas todavía.");
  });
});

describe("ProgramCoursesManager — Combobox filtering", () => {
  it("Scenario 10: shows only unassociated courses in Combobox", async () => {
    const { user } = await renderWithSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramCourses: async () => ({ programCourses: [pcMat101] }),
      listCourses: async () => ({ courses: [mat101, fis201, qui301] }),
    });

    await screen.findByText("MAT101");

    // Open combobox
    await user.click(
      screen.getByRole("combobox", { name: /agregar asignatura/i }),
    );

    // Only unassociated courses should appear
    await screen.findByText(/FIS201/);
    expect(screen.getByText(/QUI301/)).toBeInTheDocument();
    expect(screen.queryByText(/MAT101.*Cálculo/)).not.toBeInTheDocument();
  });
});

describe("ProgramCoursesManager — Add course success", () => {
  it("Scenario 12: selects course, calls addCourseToProgram, shows success toast, Combobox resets", async () => {
    const addCourseToProgram = vi.fn(async () =>
      create(ProgramCourseSchema, {
        programId: "prog-1",
        courseId: "course-fis201",
      }),
    );
    let listCallCount = 0;
    const listProgramCourses = vi.fn(async () => {
      listCallCount++;
      if (listCallCount === 1) return { programCourses: [pcMat101] };
      return {
        programCourses: [
          pcMat101,
          create(ProgramCourseSchema, {
            programId: "prog-1",
            courseId: "course-fis201",
            course: fis201,
          }),
        ],
      };
    });

    const { user } = await renderWithSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramCourses,
      listCourses: async () => ({ courses: [mat101, fis201] }),
      addCourseToProgram,
    });

    await screen.findByText("MAT101");

    await user.click(
      screen.getByRole("combobox", { name: /agregar asignatura/i }),
    );
    await screen.findByText(/FIS201/);
    await user.click(screen.getByText(/FIS201/));

    await waitFor(() =>
      expect(addCourseToProgram).toHaveBeenCalledWith(
        expect.objectContaining({
          programId: "prog-1",
          courseId: "course-fis201",
        }),
        expect.anything(),
      ),
    );
    await waitFor(() =>
      expect(toastSuccess).toHaveBeenCalledWith("Asignatura agregada."),
    );
    // Combobox should close/reset after successful add
    await waitFor(() =>
      expect(screen.queryByRole("listbox")).not.toBeInTheDocument(),
    );
  });
});

describe("ProgramCoursesManager — Add course AlreadyExists", () => {
  it("Scenario 13: shows AlreadyExists error toast", async () => {
    const addCourseToProgram = vi.fn(async () => {
      throw new ConnectError("duplicate", Code.AlreadyExists);
    });

    const { user } = await renderWithSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramCourses: async () => ({ programCourses: [pcMat101] }),
      listCourses: async () => ({ courses: [mat101, fis201] }),
      addCourseToProgram,
    });

    await screen.findByText("MAT101");
    await user.click(
      screen.getByRole("combobox", { name: /agregar asignatura/i }),
    );
    await screen.findByText(/FIS201/);
    await user.click(screen.getByText(/FIS201/));

    await waitFor(() =>
      expect(toastError).toHaveBeenCalledWith(
        "Esta asignatura ya está en la carrera.",
      ),
    );
  });
});

describe("ProgramCoursesManager — Add course InvalidArgument", () => {
  it("Scenario 14: shows generic error toast for InvalidArgument", async () => {
    const addCourseToProgram = vi.fn(async () => {
      throw new ConnectError("bad", Code.InvalidArgument);
    });

    const { user } = await renderWithSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramCourses: async () => ({ programCourses: [pcMat101] }),
      listCourses: async () => ({ courses: [mat101, fis201] }),
      addCourseToProgram,
    });

    await screen.findByText("MAT101");
    await user.click(
      screen.getByRole("combobox", { name: /agregar asignatura/i }),
    );
    await screen.findByText(/FIS201/);
    await user.click(screen.getByText(/FIS201/));

    await waitFor(() =>
      expect(toastError).toHaveBeenCalledWith(
        "No se pudo completar la acción. Inténtalo de nuevo.",
      ),
    );
  });
});

describe("ProgramCoursesManager — Remove course confirm + success", () => {
  it("Scenario 15: confirm triggers mutation and shows success toast", async () => {
    const removeCourseFromProgram = vi.fn(async () =>
      create(RemoveCourseFromProgramResponseSchema, {}),
    );

    const { user } = await renderWithSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramCourses: async () => ({ programCourses: [pcMat101] }),
      listCourses: async () => ({ courses: [mat101] }),
      removeCourseFromProgram,
    });

    await screen.findByText("MAT101");

    await user.click(
      screen.getByRole("button", { name: "Quitar MAT101 de la carrera" }),
    );
    await screen.findByRole("alertdialog");

    expect(screen.getByText(/Quitar MAT101 de la carrera/)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Quitar" }));

    await waitFor(() =>
      expect(removeCourseFromProgram).toHaveBeenCalledWith(
        expect.objectContaining({
          programId: "prog-1",
          courseId: "course-mat101",
        }),
        expect.anything(),
      ),
    );
    await waitFor(() =>
      expect(toastSuccess).toHaveBeenCalledWith("Asignatura quitada."),
    );
    await waitFor(() =>
      expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument(),
    );
  });
});

describe("ProgramCoursesManager — Remove course cancel", () => {
  it("Scenario 16: cancel closes AlertDialog without calling mutation", async () => {
    const removeCourseFromProgram = vi.fn();

    const { user } = await renderWithSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramCourses: async () => ({ programCourses: [pcMat101] }),
      listCourses: async () => ({ courses: [mat101] }),
      removeCourseFromProgram,
    });

    await screen.findByText("MAT101");
    await user.click(
      screen.getByRole("button", { name: "Quitar MAT101 de la carrera" }),
    );
    await screen.findByRole("alertdialog");

    await user.click(screen.getByRole("button", { name: "Cancelar" }));

    await waitFor(() =>
      expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument(),
    );
    expect(removeCourseFromProgram).not.toHaveBeenCalled();
  });
});

describe("ProgramCoursesManager — Remove course NotFound (stale race)", () => {
  it("Scenario 17: NotFound shows recoverable toast and closes dialog", async () => {
    const removeCourseFromProgram = vi.fn(async () => {
      throw new ConnectError("not found", Code.NotFound);
    });

    const { user } = await renderWithSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramCourses: async () => ({ programCourses: [pcMat101] }),
      listCourses: async () => ({ courses: [mat101] }),
      removeCourseFromProgram,
    });

    await screen.findByText("MAT101");
    await user.click(
      screen.getByRole("button", { name: "Quitar MAT101 de la carrera" }),
    );
    await screen.findByRole("alertdialog");

    await user.click(screen.getByRole("button", { name: "Quitar" }));

    await waitFor(() =>
      expect(toastError).toHaveBeenCalledWith(
        "Esta asignatura ya no está en la carrera.",
      ),
    );
    await waitFor(() =>
      expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument(),
    );
  });
});

describe("ProgramCoursesManager — in-flight pending disables controls", () => {
  it("Scenario 18: pending add disables Combobox trigger and [X] remove buttons", async () => {
    let resolveAdd!: () => void;
    const addCourseToProgram = vi.fn(
      () =>
        new Promise<never>((resolve) => {
          resolveAdd = () =>
            resolve(
              create(ProgramCourseSchema, {
                programId: "prog-1",
                courseId: "course-fis201",
              }) as never,
            );
        }),
    );

    const { user } = await renderWithSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramCourses: async () => ({ programCourses: [pcMat101] }),
      listCourses: async () => ({ courses: [mat101, fis201] }),
      addCourseToProgram,
    });

    await screen.findByText("MAT101");

    // Select a course to trigger the add mutation
    await user.click(
      screen.getByRole("combobox", { name: /agregar asignatura/i }),
    );
    await screen.findByText(/FIS201/);
    await user.click(screen.getByText(/FIS201/));

    // While add is in-flight, controls should be disabled
    await waitFor(() =>
      expect(
        screen.getByRole("combobox", { name: "Agregar asignatura" }),
      ).toBeDisabled(),
    );
    expect(
      screen.getByRole("button", { name: "Quitar MAT101 de la carrera" }),
    ).toBeDisabled();

    // Resolve the pending mutation
    resolveAdd();
  });
});

describe("ProgramCoursesManager — remove in-flight disables controls", () => {
  it("Scenario 19: remove in-flight disables combobox trigger and Cancelar button", async () => {
    const removeCourseFromProgram = vi.fn(
      // biome-ignore lint/suspicious/noEmptyBlockStatements: intentional never-resolving promise for testing in-flight state
      () => new Promise<never>(() => {}),
    );

    const { user } = await renderWithSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramCourses: async () => ({ programCourses: [pcMat101] }),
      listCourses: async () => ({ courses: [mat101] }),
      removeCourseFromProgram,
    });

    await screen.findByText("MAT101");

    // Open the confirm dialog
    await user.click(
      screen.getByRole("button", { name: "Quitar MAT101 de la carrera" }),
    );
    await screen.findByRole("alertdialog");

    // Click Quitar to start the in-flight remove
    await user.click(screen.getByRole("button", { name: "Quitar" }));

    // While remove is in-flight, combobox trigger (behind the modal, aria-hidden)
    // and Cancelar button (inside the dialog) must be disabled.
    await waitFor(() =>
      expect(
        screen.getByRole("combobox", {
          name: "Agregar asignatura",
          hidden: true,
        }),
      ).toBeDisabled(),
    );
    expect(screen.getByRole("button", { name: "Cancelar" })).toBeDisabled();
  });
});

describe("ProgramCoursesManager — transport/unknown error", () => {
  it("Scenario 20: unknown error code shows generic retry toast without backend details", async () => {
    const addCourseToProgram = vi.fn(async () => {
      throw new ConnectError("internal server error", Code.Internal);
    });

    const { user } = await renderWithSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramCourses: async () => ({ programCourses: [pcMat101] }),
      listCourses: async () => ({ courses: [mat101, fis201] }),
      addCourseToProgram,
    });

    await screen.findByText("MAT101");
    await user.click(
      screen.getByRole("combobox", { name: /agregar asignatura/i }),
    );
    await screen.findByText(/FIS201/);
    await user.click(screen.getByText(/FIS201/));

    await waitFor(() =>
      expect(toastError).toHaveBeenCalledWith(
        "No se pudo completar la acción. Inténtalo de nuevo.",
      ),
    );
    // No backend internals in any rendered text
    expect(screen.queryByText(/Internal/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Code\./)).not.toBeInTheDocument();
  });
});
