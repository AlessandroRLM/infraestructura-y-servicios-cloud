import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import {
  CatalogService,
  DeleteProgramResponseSchema,
  ProgramSchema,
} from "@/gen/catalog/v1/catalog_pb";
import { renderWithProviders } from "@/test";

const { toastSuccess, toastError } = vi.hoisted(() => ({
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));
vi.mock("sonner", () => ({
  toast: { success: toastSuccess, error: toastError },
}));

const mockProgram = create(ProgramSchema, {
  id: "prog-1",
  code: "ING-01",
  name: "Ingeniería de Software",
  createdAt: "2024-01-01",
  updatedAt: "2024-01-01",
});

type CatalogImpl = Partial<ServiceImpl<typeof CatalogService>>;

async function renderWithProgram(handlers: CatalogImpl) {
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
}

describe("DeleteProgramDialog", () => {
  it("S-12: cancel closes dialog without calling mutation", async () => {
    const user = userEvent.setup();
    const deleteProgram = vi.fn(async () =>
      create(DeleteProgramResponseSchema, {}),
    );

    await renderWithProgram({
      listPrograms: async () => ({ programs: [mockProgram] }),
      deleteProgram,
    });

    const deleteButtons = screen.getAllByRole("button", { name: /eliminar/i });
    await user.click(deleteButtons[0]);
    await screen.findByRole("alertdialog");

    await user.click(screen.getByRole("button", { name: "Cancelar" }));
    await waitFor(() =>
      expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument(),
    );
    expect(deleteProgram).not.toHaveBeenCalled();
  });

  it("S-13: success closes dialog and shows success toast", async () => {
    const user = userEvent.setup();
    const deleteProgram = vi.fn(async () =>
      create(DeleteProgramResponseSchema, {}),
    );

    await renderWithProgram({
      listPrograms: async () => ({ programs: [mockProgram] }),
      deleteProgram,
    });

    const deleteButtons = screen.getAllByRole("button", { name: /eliminar/i });
    await user.click(deleteButtons[0]);
    await screen.findByRole("alertdialog");

    await user.click(screen.getByRole("button", { name: "Eliminar" }));

    await waitFor(() =>
      expect(toastSuccess).toHaveBeenCalledWith("Carrera eliminada"),
    );
    await waitFor(() =>
      expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument(),
    );
  });

  it("S-14: FailedPrecondition shows in-dialog message, no toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const deleteProgram = vi.fn(async () => {
      throw new ConnectError("has dependents", Code.FailedPrecondition);
    });

    await renderWithProgram({
      listPrograms: async () => ({ programs: [mockProgram] }),
      deleteProgram,
    });

    const deleteButtons = screen.getAllByRole("button", { name: /eliminar/i });
    await user.click(deleteButtons[0]);
    await screen.findByRole("alertdialog");

    await user.click(screen.getByRole("button", { name: "Eliminar" }));

    await waitFor(() =>
      expect(
        screen.getByText(
          /No se puede eliminar: la carrera tiene asignaturas o cupos asociados\. Quita esas asociaciones primero\./,
        ),
      ).toBeInTheDocument(),
    );
    expect(toastError).not.toHaveBeenCalled();
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
  });

  it("S-15: other transport error shows in-dialog error, no toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const deleteProgram = vi.fn(async () => {
      throw new ConnectError("internal", Code.Internal);
    });

    await renderWithProgram({
      listPrograms: async () => ({ programs: [mockProgram] }),
      deleteProgram,
    });

    const deleteButtons = screen.getAllByRole("button", { name: /eliminar/i });
    await user.click(deleteButtons[0]);
    await screen.findByRole("alertdialog");

    await user.click(screen.getByRole("button", { name: "Eliminar" }));

    await waitFor(() =>
      expect(
        screen.getByText(/No se pudo eliminar la carrera/),
      ).toBeInTheDocument(),
    );
    expect(toastError).not.toHaveBeenCalled();
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
  });
});
