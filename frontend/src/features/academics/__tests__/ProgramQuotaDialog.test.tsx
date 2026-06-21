/**
 * ProgramQuotaDialog — tested via the per-program Sheet (Cupos tab).
 * programId is always injected by the Sheet context — there is no program
 * selector in the form (create or edit mode).
 */
import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import type { AuthenticatedSession } from "@/features/auth";
import {
  CatalogService,
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

/** Navigates to the Cupos tab via the Sheet. */
async function openQuotasTab(handlers: CatalogImpl) {
  const user = userEvent.setup();
  renderWithProviders({
    route: "/admin/academics",
    transport: makeStubTransport([CatalogService, handlers]),
    session: { status: "authenticated", ...adminSession },
    sessionSource: adminSessionSource,
  });
  await screen.findByRole("heading", { name: "Académico" });
  await screen.findByText("ING");
  await user.click(screen.getByRole("button", { name: "Acciones ING" }));
  await user.click(screen.getByRole("menuitem", { name: /gestionar/i }));
  await user.click(screen.getByRole("tab", { name: "Cupos" }));
  return { user };
}

describe("ProgramQuotaDialog — create mode", () => {
  beforeEach(() => {
    toastSuccess.mockClear();
    toastError.mockClear();
  });

  it("success closes dialog, shows success toast, sends correct programId", async () => {
    const createProgramQuota = vi.fn(async () => mockQuota);
    const listProgramQuotas = vi.fn(async () => ({ programQuotas: [] }));

    const { user } = await openQuotasTab({
      createProgramQuota,
      listProgramQuotas,
      listPrograms: async () => ({
        programs: [mockProgram],
        nextPageToken: "",
      }),
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    await screen.findByText("Todavía no hay cupos");

    // Open create dialog from the header Crear cupo button
    const createButtons = screen.getAllByRole("button", {
      name: /crear cupo/i,
    });
    await user.click(createButtons[0]);
    await screen.findByRole("dialog");

    const dialog = screen.getByRole("dialog");

    // No program combobox — programId is injected from context
    expect(within(dialog).queryByRole("combobox")).not.toBeInTheDocument();

    await user.type(within(dialog).getByLabelText("Año"), "2025");
    await user.type(within(dialog).getByLabelText("Cupo"), "60");
    await user.click(
      within(dialog).getByRole("button", { name: /crear cupo/i }),
    );

    await waitFor(() => expect(createProgramQuota).toHaveBeenCalledTimes(1));
    const firstCall = createProgramQuota.mock.calls[0] as unknown as [
      { programId: string; year: number; admissionQuota: number },
    ];
    expect(firstCall[0]).toMatchObject({
      programId: "program-1",
      year: 2025,
      admissionQuota: 60,
    });

    await waitFor(() =>
      expect(toastSuccess).toHaveBeenCalledWith("Cupo creado"),
    );
    // The create Dialog closes; the Sheet (also role=dialog) stays open
    await waitFor(() =>
      expect(
        screen.queryByRole("dialog", { name: /crear cupo/i }),
      ).not.toBeInTheDocument(),
    );
  });

  it("transport error shows toast, dialog stays open", async () => {
    const createProgramQuota = vi.fn(async () => {
      throw new ConnectError("internal", Code.Internal);
    });
    const listProgramQuotas = vi.fn(async () => ({ programQuotas: [] }));

    const { user } = await openQuotasTab({
      createProgramQuota,
      listProgramQuotas,
      listPrograms: async () => ({
        programs: [mockProgram],
        nextPageToken: "",
      }),
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    await screen.findByText("Todavía no hay cupos");
    const createButtons = screen.getAllByRole("button", {
      name: /crear cupo/i,
    });
    await user.click(createButtons[0]);
    await screen.findByRole("dialog");

    const dialog = screen.getByRole("dialog");
    await user.type(within(dialog).getByLabelText("Año"), "2025");
    await user.type(within(dialog).getByLabelText("Cupo"), "60");
    await user.click(
      within(dialog).getByRole("button", { name: /crear cupo/i }),
    );

    await waitFor(() => expect(toastError).toHaveBeenCalled());
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("form validation: submitting empty form shows field errors, no RPC call", async () => {
    const createProgramQuota = vi.fn();
    const listProgramQuotas = vi.fn(async () => ({ programQuotas: [] }));

    const { user } = await openQuotasTab({
      createProgramQuota,
      listProgramQuotas,
      listPrograms: async () => ({
        programs: [mockProgram],
        nextPageToken: "",
      }),
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    await screen.findByText("Todavía no hay cupos");
    const createButtons = screen.getAllByRole("button", {
      name: /crear cupo/i,
    });
    await user.click(createButtons[0]);
    await screen.findByRole("dialog");

    const dialog = screen.getByRole("dialog");
    await user.click(
      within(dialog).getByRole("button", { name: /crear cupo/i }),
    );

    // Year and quota fields should show errors
    const errors = await screen.findAllByRole("alert");
    expect(errors.length).toBeGreaterThanOrEqual(1);
    expect(createProgramQuota).not.toHaveBeenCalled();
  });
});

describe("ProgramQuotaDialog — edit mode", () => {
  beforeEach(() => {
    toastSuccess.mockClear();
    toastError.mockClear();
  });

  it("pre-fills year and quota; no program selector (programId immutable); calls updateProgramQuota", async () => {
    const updateProgramQuota = vi.fn(async () => mockQuota);
    const listProgramQuotas = vi.fn(async () => ({
      programQuotas: [mockQuota],
    }));

    const { user } = await openQuotasTab({
      updateProgramQuota,
      listProgramQuotas,
      listPrograms: async () => ({
        programs: [mockProgram],
        nextPageToken: "",
      }),
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    await screen.findByText("60");

    const editButton = screen.getByRole("button", {
      name: `Editar cupo ${mockQuota.id}`,
    });
    await user.click(editButton);
    await screen.findByRole("dialog");

    const dialog = screen.getByRole("dialog");

    // No program selector
    expect(within(dialog).queryByRole("combobox")).not.toBeInTheDocument();

    const yearInput = within(dialog).getByLabelText("Año") as HTMLInputElement;
    expect(yearInput.value).toBe("2025");

    const quotaInput = within(dialog).getByLabelText(
      "Cupo",
    ) as HTMLInputElement;
    expect(quotaInput.value).toBe("60");

    await user.clear(quotaInput);
    await user.type(quotaInput, "80");
    await user.click(
      within(dialog).getByRole("button", { name: /guardar cambios/i }),
    );

    await waitFor(() => expect(updateProgramQuota).toHaveBeenCalledTimes(1));
    const firstCall = updateProgramQuota.mock.calls[0] as unknown as [
      { id: string; year: number; admissionQuota: number },
    ];
    expect(firstCall[0]).toMatchObject({
      id: "quota-1",
      admissionQuota: 80,
    });
    expect(typeof firstCall[0].admissionQuota).toBe("number");

    await waitFor(() =>
      expect(toastSuccess).toHaveBeenCalledWith("Cupo actualizado"),
    );
  });

  it("transport error on update shows toast, dialog stays open", async () => {
    const updateProgramQuota = vi.fn(async () => {
      throw new ConnectError("internal", Code.Internal);
    });
    const listProgramQuotas = vi.fn(async () => ({
      programQuotas: [mockQuota],
    }));

    const { user } = await openQuotasTab({
      updateProgramQuota,
      listProgramQuotas,
      listPrograms: async () => ({
        programs: [mockProgram],
        nextPageToken: "",
      }),
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    await screen.findByText("60");

    await user.click(
      screen.getByRole("button", { name: `Editar cupo ${mockQuota.id}` }),
    );
    await screen.findByRole("dialog");

    const dialog = screen.getByRole("dialog");
    await user.click(
      within(dialog).getByRole("button", { name: /guardar cambios/i }),
    );

    await waitFor(() => expect(toastError).toHaveBeenCalled());
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });
});
