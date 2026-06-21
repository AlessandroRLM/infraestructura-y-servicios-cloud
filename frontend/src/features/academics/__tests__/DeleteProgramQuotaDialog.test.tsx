/**
 * DeleteProgramQuotaDialog — tested via the per-program Sheet (Cupos tab).
 */
import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import type { AuthenticatedSession } from "@/features/auth";
import {
  CatalogService,
  DeleteProgramQuotaResponseSchema,
  ProgramQuotaSchema,
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

type CatalogImpl = Partial<ServiceImpl<typeof CatalogService>>;

const mockProgram = create(ProgramSchema, {
  id: "program-1",
  code: "ING",
  name: "Ingeniería Civil",
  createdAt: "2024-01-01",
  updatedAt: "2024-01-01",
});

const mockQuota = create(ProgramQuotaSchema, {
  id: "quota-1",
  programId: "program-1",
  year: 2025,
  admissionQuota: 60,
  createdAt: "2024-01-01",
  updatedAt: "2024-01-01",
});

const adminSession: AuthenticatedSession = {
  userId: "1",
  email: "admin@test.com",
  roles: ["admin"],
  permissions: ["catalog.manage"],
};

const adminSessionSource = {
  getSession: async (): Promise<AuthenticatedSession> => adminSession,
};

/** Opens the Cupos tab via the Sheet and waits for the quota row to render. */
async function renderWithQuota(handlers: CatalogImpl) {
  const user = userEvent.setup();
  renderWithProviders({
    route: "/admin/academics",
    transport: makeStubTransport([CatalogService, handlers]),
    session: { status: "authenticated", ...adminSession },
    sessionSource: adminSessionSource,
  });
  await screen.findByText("ING");
  await user.click(screen.getByRole("button", { name: "Acciones ING" }));
  await user.click(screen.getByRole("menuitem", { name: /gestionar/i }));
  await user.click(screen.getByRole("tab", { name: "Cupos" }));
  await screen.findByText("60");
  return { user };
}

describe("DeleteProgramQuotaDialog", () => {
  it("cancel closes dialog without calling mutation", async () => {
    const deleteProgramQuota = vi.fn(async () =>
      create(DeleteProgramQuotaResponseSchema, {}),
    );

    const { user } = await renderWithQuota({
      listProgramQuotas: async () => ({ programQuotas: [mockQuota] }),
      listPrograms: async () => ({
        programs: [mockProgram],
        nextPageToken: "",
      }),
      deleteProgramQuota,
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    const deleteButtons = screen.getAllByRole("button", {
      name: /eliminar cupo/i,
    });
    await user.click(deleteButtons[0]);
    await screen.findByRole("alertdialog");

    await user.click(screen.getByRole("button", { name: "Cancelar" }));
    await waitFor(() =>
      expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument(),
    );
    expect(deleteProgramQuota).not.toHaveBeenCalled();
  });

  it("success closes dialog and shows success toast", async () => {
    const deleteProgramQuota = vi.fn(async () =>
      create(DeleteProgramQuotaResponseSchema, {}),
    );

    const { user } = await renderWithQuota({
      listProgramQuotas: async () => ({ programQuotas: [mockQuota] }),
      listPrograms: async () => ({
        programs: [mockProgram],
        nextPageToken: "",
      }),
      deleteProgramQuota,
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    const deleteButtons = screen.getAllByRole("button", {
      name: /eliminar cupo/i,
    });
    await user.click(deleteButtons[0]);
    await screen.findByRole("alertdialog");

    await user.click(screen.getByRole("button", { name: "Eliminar" }));

    await waitFor(() =>
      expect(toastSuccess).toHaveBeenCalledWith("Cupo eliminado"),
    );
    await waitFor(() =>
      expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument(),
    );
  });

  it("FailedPrecondition shows in-dialog message, no toast, dialog stays open", async () => {
    const deleteProgramQuota = vi.fn(async () => {
      throw new ConnectError("has dependents", Code.FailedPrecondition);
    });

    const { user } = await renderWithQuota({
      listProgramQuotas: async () => ({ programQuotas: [mockQuota] }),
      listPrograms: async () => ({
        programs: [mockProgram],
        nextPageToken: "",
      }),
      deleteProgramQuota,
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    const deleteButtons = screen.getAllByRole("button", {
      name: /eliminar cupo/i,
    });
    await user.click(deleteButtons[0]);
    await screen.findByRole("alertdialog");

    await user.click(screen.getByRole("button", { name: "Eliminar" }));

    await waitFor(() =>
      expect(screen.getByText(/el cupo está en uso/)).toBeInTheDocument(),
    );
    expect(screen.queryByText(/FailedPrecondition/)).not.toBeInTheDocument();
    expect(toastError).not.toHaveBeenCalled();
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
  });

  it("transport error shows in-dialog message, no toast, dialog stays open", async () => {
    const deleteProgramQuota = vi.fn(async () => {
      throw new ConnectError("internal", Code.Internal);
    });

    const { user } = await renderWithQuota({
      listProgramQuotas: async () => ({ programQuotas: [mockQuota] }),
      listPrograms: async () => ({
        programs: [mockProgram],
        nextPageToken: "",
      }),
      deleteProgramQuota,
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    const deleteButtons = screen.getAllByRole("button", {
      name: /eliminar cupo/i,
    });
    await user.click(deleteButtons[0]);
    await screen.findByRole("alertdialog");

    await user.click(screen.getByRole("button", { name: "Eliminar" }));

    await waitFor(() =>
      expect(
        screen.getByText(/No se pudo eliminar el cupo/),
      ).toBeInTheDocument(),
    );
    expect(toastError).not.toHaveBeenCalled();
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
  });
});
