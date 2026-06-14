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
  DeleteCourseResponseSchema,
} from "@/gen/catalog/v1/catalog_pb";
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

async function renderWithCourse(handlers: CatalogImpl) {
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
  await screen.findByText("CS-101");
}

describe("DeleteCourseDialog", () => {
  it("SC-21: cancel closes dialog without calling mutation", async () => {
    const user = userEvent.setup();
    const deleteCourse = vi.fn(async () =>
      create(DeleteCourseResponseSchema, {}),
    );

    await renderWithCourse({
      listCourses: async () => ({ courses: [mockCourse] }),
      deleteCourse,
    });

    const deleteButtons = screen.getAllByRole("button", { name: /eliminar/i });
    await user.click(deleteButtons[0]);
    await screen.findByRole("alertdialog");

    await user.click(screen.getByRole("button", { name: "Cancelar" }));
    await waitFor(() =>
      expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument(),
    );
    expect(deleteCourse).not.toHaveBeenCalled();
  });

  it("SC-22: success closes dialog and shows success toast", async () => {
    const user = userEvent.setup();
    const deleteCourse = vi.fn(async () =>
      create(DeleteCourseResponseSchema, {}),
    );

    await renderWithCourse({
      listCourses: async () => ({ courses: [mockCourse] }),
      deleteCourse,
    });

    const deleteButtons = screen.getAllByRole("button", { name: /eliminar/i });
    await user.click(deleteButtons[0]);
    await screen.findByRole("alertdialog");

    await user.click(screen.getByRole("button", { name: "Eliminar" }));

    await waitFor(() =>
      expect(toastSuccess).toHaveBeenCalledWith("Asignatura eliminada"),
    );
    await waitFor(() =>
      expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument(),
    );
  });

  it("SC-23: FailedPrecondition shows in-dialog message, no toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const deleteCourse = vi.fn(async () => {
      throw new ConnectError("has dependents", Code.FailedPrecondition);
    });

    await renderWithCourse({
      listCourses: async () => ({ courses: [mockCourse] }),
      deleteCourse,
    });

    const deleteButtons = screen.getAllByRole("button", { name: /eliminar/i });
    await user.click(deleteButtons[0]);
    await screen.findByRole("alertdialog");

    await user.click(screen.getByRole("button", { name: "Eliminar" }));

    await waitFor(() =>
      expect(screen.getByText(/la asignatura está en uso/)).toBeInTheDocument(),
    );
    expect(screen.queryByText(/FailedPrecondition/)).not.toBeInTheDocument();
    expect(toastError).not.toHaveBeenCalled();
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
  });

  it("SC-24: transport error shows in-dialog message, no toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const deleteCourse = vi.fn(async () => {
      throw new ConnectError("internal", Code.Internal);
    });

    await renderWithCourse({
      listCourses: async () => ({ courses: [mockCourse] }),
      deleteCourse,
    });

    const deleteButtons = screen.getAllByRole("button", { name: /eliminar/i });
    await user.click(deleteButtons[0]);
    await screen.findByRole("alertdialog");

    await user.click(screen.getByRole("button", { name: "Eliminar" }));

    await waitFor(() =>
      expect(
        screen.getByText(/No se pudo eliminar la asignatura/),
      ).toBeInTheDocument(),
    );
    expect(toastError).not.toHaveBeenCalled();
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
  });
});
