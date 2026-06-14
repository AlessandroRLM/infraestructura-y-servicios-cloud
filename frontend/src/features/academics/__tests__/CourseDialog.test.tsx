import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import { CatalogService, CourseSchema } from "@/gen/catalog/v1/catalog_pb";
import { renderWithProviders } from "@/test";

const { toastSuccess, toastError } = vi.hoisted(() => ({
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));
vi.mock("sonner", () => ({
  toast: { success: toastSuccess, error: toastError },
}));

const mockCourse = create(CourseSchema, {
  id: "course-1",
  code: "CS-101",
  name: "Cálculo",
  credits: 5,
  createdAt: "2024-01-01",
  updatedAt: "2024-01-01",
});

type CatalogImpl = Partial<ServiceImpl<typeof CatalogService>>;

async function renderCoursesPage(handlers: CatalogImpl = {}) {
  renderWithProviders({
    route: "/academics?tab=courses",
    transport: makeStubTransport([CatalogService, handlers]),
    session: {
      status: "authenticated",
      userId: "1",
      email: "admin@test.com",
      roles: ["admin"],
      permissions: ["catalog.manage"],
    },
  });
  await screen.findByRole("heading", { name: "Académico" });
}

describe("CourseDialog — create mode", () => {
  beforeEach(() => {
    toastSuccess.mockClear();
    toastError.mockClear();
  });

  it("success closes dialog, shows success toast, invalidates list", async () => {
    const user = userEvent.setup();
    const createCourse = vi.fn(async () => mockCourse);
    const listCourses = vi.fn(async () => ({ courses: [] }));

    await renderCoursesPage({ createCourse, listCourses });

    // With empty courses, both header and empty-state buttons render.
    const openButtons = screen.getAllByRole("button", {
      name: /crear asignatura/i,
    });
    await user.click(openButtons[0]);
    await screen.findByRole("dialog");

    const dialog = screen.getByRole("dialog");
    await user.type(within(dialog).getByLabelText("Código"), "CS-201");
    await user.type(within(dialog).getByLabelText("Nombre"), "Álgebra Lineal");

    // Select credits
    const trigger = within(dialog).getByRole("combobox");
    await user.click(trigger);
    await user.click(screen.getByRole("option", { name: "6" }));

    const submitBtn = within(dialog).getByRole("button", {
      name: /crear asignatura/i,
    });
    await user.click(submitBtn);

    await waitFor(() => expect(createCourse).toHaveBeenCalledTimes(1));
    const firstCreateCall = createCourse.mock.calls[0] as unknown as [
      { code: string; name: string; credits: number },
    ];
    expect(firstCreateCall[0].credits).toBe(6);
    expect(typeof firstCreateCall[0].credits).toBe("number");

    await waitFor(() =>
      expect(toastSuccess).toHaveBeenCalledWith("Asignatura creada"),
    );
    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
    );
  });

  it("AlreadyExists shows inline code error, no toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const createCourse = vi.fn(async () => {
      throw new ConnectError("duplicate", Code.AlreadyExists);
    });
    const listCourses = vi.fn(async () => ({ courses: [] }));

    await renderCoursesPage({ createCourse, listCourses });

    const openButtons = screen.getAllByRole("button", {
      name: /crear asignatura/i,
    });
    await user.click(openButtons[0]);
    await screen.findByRole("dialog");

    const dialog = screen.getByRole("dialog");
    await user.type(within(dialog).getByLabelText("Código"), "DUP-01");
    await user.type(within(dialog).getByLabelText("Nombre"), "Duplicado");

    const trigger = within(dialog).getByRole("combobox");
    await user.click(trigger);
    await user.click(screen.getByRole("option", { name: "4" }));

    const submitBtn = within(dialog).getByRole("button", {
      name: /crear asignatura/i,
    });
    await user.click(submitBtn);

    await waitFor(() =>
      expect(
        screen.getByText("Ya existe una asignatura con ese código"),
      ).toBeInTheDocument(),
    );
    expect(toastError).not.toHaveBeenCalled();
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("transport error shows toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const createCourse = vi.fn(async () => {
      throw new ConnectError("internal", Code.Internal);
    });
    const listCourses = vi.fn(async () => ({ courses: [] }));

    await renderCoursesPage({ createCourse, listCourses });

    const openButtons = screen.getAllByRole("button", {
      name: /crear asignatura/i,
    });
    await user.click(openButtons[0]);
    await screen.findByRole("dialog");

    const dialog = screen.getByRole("dialog");
    await user.type(within(dialog).getByLabelText("Código"), "OK-01");
    await user.type(within(dialog).getByLabelText("Nombre"), "Nombre válido");

    const trigger = within(dialog).getByRole("combobox");
    await user.click(trigger);
    await user.click(screen.getByRole("option", { name: "5" }));

    const submitBtn = within(dialog).getByRole("button", {
      name: /crear asignatura/i,
    });
    await user.click(submitBtn);

    await waitFor(() => expect(toastError).toHaveBeenCalled());
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });
});

describe("CourseDialog — edit mode", () => {
  beforeEach(() => {
    toastSuccess.mockClear();
    toastError.mockClear();
  });

  it("SC-18/SC-19: edit mode pre-fills all fields and calls updateCourse with id + credits as number", async () => {
    const user = userEvent.setup();
    const updateCourse = vi.fn(async () => mockCourse);
    const listCourses = vi.fn(async () => ({ courses: [mockCourse] }));

    await renderCoursesPage({ updateCourse, listCourses });

    await screen.findByText("CS-101");

    const editButtons = screen.getAllByRole("button", { name: /editar/i });
    await user.click(editButtons[0]);

    await screen.findByRole("dialog");
    const dialog = screen.getByRole("dialog");

    const codeInput = within(dialog).getByLabelText(
      "Código",
    ) as HTMLInputElement;
    const nameInput = within(dialog).getByLabelText(
      "Nombre",
    ) as HTMLInputElement;
    expect(codeInput.value).toBe("CS-101");
    expect(nameInput.value).toBe("Cálculo");

    // credits combobox should show "5" (preselected)
    const trigger = within(dialog).getByRole("combobox");
    expect(trigger).toHaveTextContent("5");

    await user.clear(nameInput);
    await user.type(nameInput, "Cálculo I");
    await user.click(
      within(dialog).getByRole("button", { name: /guardar cambios/i }),
    );

    await waitFor(() => expect(updateCourse).toHaveBeenCalledTimes(1));
    const firstUpdateCall = updateCourse.mock.calls[0] as unknown as [
      { id: string; code: string; name: string; credits: number },
    ];
    expect(firstUpdateCall[0]).toMatchObject({
      id: "course-1",
      code: "CS-101",
      name: "Cálculo I",
      credits: 5,
    });
    expect(typeof firstUpdateCall[0].credits).toBe("number");

    await waitFor(() =>
      expect(toastSuccess).toHaveBeenCalledWith("Asignatura actualizada"),
    );
  });

  it("AlreadyExists on update shows inline code error, no toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const updateCourse = vi.fn(async () => {
      throw new ConnectError("duplicate", Code.AlreadyExists);
    });
    const listCourses = vi.fn(async () => ({ courses: [mockCourse] }));

    await renderCoursesPage({ updateCourse, listCourses });
    await screen.findByText("CS-101");

    const editButtons = screen.getAllByRole("button", { name: /editar/i });
    await user.click(editButtons[0]);
    await screen.findByRole("dialog");

    const dialog = screen.getByRole("dialog");
    await user.click(
      within(dialog).getByRole("button", { name: /guardar cambios/i }),
    );

    await waitFor(() =>
      expect(
        screen.getByText("Ya existe una asignatura con ese código"),
      ).toBeInTheDocument(),
    );
    expect(toastError).not.toHaveBeenCalled();
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("transport error on update shows toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const updateCourse = vi.fn(async () => {
      throw new ConnectError("internal", Code.Internal);
    });
    const listCourses = vi.fn(async () => ({ courses: [mockCourse] }));

    await renderCoursesPage({ updateCourse, listCourses });
    await screen.findByText("CS-101");

    const editButtons = screen.getAllByRole("button", { name: /editar/i });
    await user.click(editButtons[0]);
    await screen.findByRole("dialog");

    const dialog = screen.getByRole("dialog");
    await user.click(
      within(dialog).getByRole("button", { name: /guardar cambios/i }),
    );

    await waitFor(() => expect(toastError).toHaveBeenCalled());
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });
});
