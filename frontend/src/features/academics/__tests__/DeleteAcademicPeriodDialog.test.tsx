import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import {
  AcademicPeriodSchema,
  CatalogService,
  DeleteAcademicPeriodResponseSchema,
} from "@/gen/catalog/v1/catalog_pb";
import { renderWithProviders } from "@/test";

const { toastSuccess, toastError } = vi.hoisted(() => ({
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));
vi.mock("sonner", () => ({
  toast: { success: toastSuccess, error: toastError },
}));

const mockPeriod = create(AcademicPeriodSchema, {
  id: "period-1",
  year: 2025,
  term: 1,
  startDate: "2025-03-01",
  endDate: "2025-07-15",
  createdAt: "2024-12-01",
  updatedAt: "2024-12-01",
});

type CatalogImpl = Partial<ServiceImpl<typeof CatalogService>>;

async function renderWithPeriod(handlers: CatalogImpl) {
  renderWithProviders({
    route: "/admin/academics?tab=periods",
    transport: makeStubTransport([CatalogService, handlers]),
    session: {
      status: "authenticated",
      userId: "1",
      email: "admin@test.com",
      roles: ["admin"],
      permissions: ["catalog.manage"],
    },
  });
  await screen.findByText("2025-03-01");
}

describe("DeleteAcademicPeriodDialog", () => {
  it("cancel closes dialog without calling mutation", async () => {
    const user = userEvent.setup();
    const deleteAcademicPeriod = vi.fn(async () =>
      create(DeleteAcademicPeriodResponseSchema, {}),
    );

    await renderWithPeriod({
      listAcademicPeriods: async () => ({ academicPeriods: [mockPeriod] }),
      deleteAcademicPeriod,
    });

    const deleteButtons = screen.getAllByRole("button", {
      name: /eliminar período/i,
    });
    await user.click(deleteButtons[0]);
    await screen.findByRole("alertdialog");

    await user.click(screen.getByRole("button", { name: "Cancelar" }));
    await waitFor(() =>
      expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument(),
    );
    expect(deleteAcademicPeriod).not.toHaveBeenCalled();
  });

  it("success closes dialog and shows success toast", async () => {
    const user = userEvent.setup();
    const deleteAcademicPeriod = vi.fn(async () =>
      create(DeleteAcademicPeriodResponseSchema, {}),
    );

    await renderWithPeriod({
      listAcademicPeriods: async () => ({ academicPeriods: [mockPeriod] }),
      deleteAcademicPeriod,
    });

    const deleteButtons = screen.getAllByRole("button", {
      name: /eliminar período/i,
    });
    await user.click(deleteButtons[0]);
    await screen.findByRole("alertdialog");

    await user.click(screen.getByRole("button", { name: "Eliminar" }));

    await waitFor(() =>
      expect(toastSuccess).toHaveBeenCalledWith("Período eliminado"),
    );
    await waitFor(() =>
      expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument(),
    );
  });

  it("FailedPrecondition shows in-dialog message, no toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const deleteAcademicPeriod = vi.fn(async () => {
      throw new ConnectError("has dependents", Code.FailedPrecondition);
    });

    await renderWithPeriod({
      listAcademicPeriods: async () => ({ academicPeriods: [mockPeriod] }),
      deleteAcademicPeriod,
    });

    const deleteButtons = screen.getAllByRole("button", {
      name: /eliminar período/i,
    });
    await user.click(deleteButtons[0]);
    await screen.findByRole("alertdialog");

    await user.click(screen.getByRole("button", { name: "Eliminar" }));

    await waitFor(() =>
      expect(screen.getByText(/el período está en uso/)).toBeInTheDocument(),
    );
    expect(screen.queryByText(/FailedPrecondition/)).not.toBeInTheDocument();
    expect(toastError).not.toHaveBeenCalled();
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
  });

  it("transport error shows in-dialog message, no toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const deleteAcademicPeriod = vi.fn(async () => {
      throw new ConnectError("internal", Code.Internal);
    });

    await renderWithPeriod({
      listAcademicPeriods: async () => ({ academicPeriods: [mockPeriod] }),
      deleteAcademicPeriod,
    });

    const deleteButtons = screen.getAllByRole("button", {
      name: /eliminar período/i,
    });
    await user.click(deleteButtons[0]);
    await screen.findByRole("alertdialog");

    await user.click(screen.getByRole("button", { name: "Eliminar" }));

    await waitFor(() =>
      expect(
        screen.getByText(/No se pudo eliminar el período/),
      ).toBeInTheDocument(),
    );
    expect(toastError).not.toHaveBeenCalled();
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
  });
});
