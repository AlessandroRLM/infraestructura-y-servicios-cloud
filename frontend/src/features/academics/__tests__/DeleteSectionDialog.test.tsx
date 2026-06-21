import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import {
  CatalogService,
  DeleteSectionResponseSchema,
  SectionSchema,
} from "@/gen/catalog/v1/catalog_pb";
import { renderWithProviders } from "@/test";

const { toastSuccess, toastError } = vi.hoisted(() => ({
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));
vi.mock("sonner", () => ({
  toast: { success: toastSuccess, error: toastError },
}));

const mockSection = create(SectionSchema, {
  id: "section-1",
  courseId: "course-1",
  academicPeriodId: "period-1",
  seatCapacity: 30,
  createdAt: "2024-01-01",
  updatedAt: "2024-01-01",
});

type CatalogImpl = Partial<ServiceImpl<typeof CatalogService>>;

async function renderWithSection(handlers: CatalogImpl) {
  renderWithProviders({
    route: "/admin/academics?tab=sections",
    transport: makeStubTransport([CatalogService, handlers]),
    session: {
      status: "authenticated",
      userId: "1",
      email: "admin@test.com",
      roles: ["admin"],
      permissions: ["catalog.manage"],
    },
  });
  await screen.findByText("30");
}

describe("DeleteSectionDialog", () => {
  it("cancel closes dialog without calling mutation", async () => {
    const user = userEvent.setup();
    const deleteSection = vi.fn(async () =>
      create(DeleteSectionResponseSchema, {}),
    );

    await renderWithSection({
      listSections: async () => ({ sections: [mockSection] }),
      deleteSection,
    });

    const deleteButtons = screen.getAllByRole("button", {
      name: /eliminar sección/i,
    });
    await user.click(deleteButtons[0]);
    await screen.findByRole("alertdialog");

    await user.click(screen.getByRole("button", { name: "Cancelar" }));
    await waitFor(() =>
      expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument(),
    );
    expect(deleteSection).not.toHaveBeenCalled();
  });

  it("success closes dialog and shows success toast", async () => {
    const user = userEvent.setup();
    const deleteSection = vi.fn(async () =>
      create(DeleteSectionResponseSchema, {}),
    );

    await renderWithSection({
      listSections: async () => ({ sections: [mockSection] }),
      deleteSection,
    });

    const deleteButtons = screen.getAllByRole("button", {
      name: /eliminar sección/i,
    });
    await user.click(deleteButtons[0]);
    await screen.findByRole("alertdialog");

    await user.click(screen.getByRole("button", { name: "Eliminar" }));

    await waitFor(() =>
      expect(toastSuccess).toHaveBeenCalledWith("Sección eliminada"),
    );
    await waitFor(() =>
      expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument(),
    );
  });

  it("FailedPrecondition shows in-dialog message, no toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const deleteSection = vi.fn(async () => {
      throw new ConnectError("has dependents", Code.FailedPrecondition);
    });

    await renderWithSection({
      listSections: async () => ({ sections: [mockSection] }),
      deleteSection,
    });

    const deleteButtons = screen.getAllByRole("button", {
      name: /eliminar sección/i,
    });
    await user.click(deleteButtons[0]);
    await screen.findByRole("alertdialog");

    await user.click(screen.getByRole("button", { name: "Eliminar" }));

    await waitFor(() =>
      expect(screen.getByText(/la sección está en uso/)).toBeInTheDocument(),
    );
    expect(screen.queryByText(/FailedPrecondition/)).not.toBeInTheDocument();
    expect(toastError).not.toHaveBeenCalled();
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
  });

  it("transport error shows in-dialog message, no toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const deleteSection = vi.fn(async () => {
      throw new ConnectError("internal", Code.Internal);
    });

    await renderWithSection({
      listSections: async () => ({ sections: [mockSection] }),
      deleteSection,
    });

    const deleteButtons = screen.getAllByRole("button", {
      name: /eliminar sección/i,
    });
    await user.click(deleteButtons[0]);
    await screen.findByRole("alertdialog");

    await user.click(screen.getByRole("button", { name: "Eliminar" }));

    await waitFor(() =>
      expect(
        screen.getByText(/No se pudo eliminar la sección/),
      ).toBeInTheDocument(),
    );
    expect(toastError).not.toHaveBeenCalled();
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
  });
});
