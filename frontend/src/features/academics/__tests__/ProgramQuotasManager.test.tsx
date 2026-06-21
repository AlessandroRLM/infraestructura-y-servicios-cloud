import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen, waitFor, within } from "@testing-library/react";
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
  id: "prog-1",
  code: "ING-01",
  name: "Ingeniería de Software",
  createdAt: "2024-01-01",
  updatedAt: "2024-01-01",
});

const quota2024 = create(ProgramQuotaSchema, {
  id: "q1",
  programId: "prog-1",
  year: 2024,
  admissionQuota: 50,
  createdAt: "2024-01-01",
  updatedAt: "2024-01-01",
});

const quota2025 = create(ProgramQuotaSchema, {
  id: "q2",
  programId: "prog-1",
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

/** Opens the per-program Sheet and navigates to the "Cupos" tab. */
async function renderWithSheet(handlers: CatalogImpl) {
  const user = userEvent.setup();
  renderWithProviders({
    route: "/admin/academics",
    transport: makeStubTransport([CatalogService, handlers]),
    session: { status: "authenticated", ...adminSession },
    sessionSource: adminSessionSource,
  });
  // Wait for programs table to load
  await screen.findByText("ING-01");
  // Open the ⋯ dropdown and click Gestionar
  await user.click(screen.getByRole("button", { name: "Acciones ING-01" }));
  await user.click(screen.getByRole("menuitem", { name: /gestionar/i }));
  // Switch to the Cupos tab inside the Sheet
  await user.click(screen.getByRole("tab", { name: "Cupos" }));
  return { user };
}

describe("ProgramQuotasManager — via tabbed Sheet", () => {
  it("opens Sheet with Gestionar title and Cupos tab renders the manager", async () => {
    await renderWithSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramQuotas: async () => ({ programQuotas: [quota2025] }),
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    expect(
      screen.getByText("Gestionar Ingeniería de Software"),
    ).toBeInTheDocument();
    await screen.findByText("60");
    expect(screen.getByText("2025")).toBeInTheDocument();
  });

  it("calls listProgramQuotas with the correct programId (not empty)", async () => {
    const listProgramQuotas = vi.fn(async () => ({ programQuotas: [] }));
    await renderWithSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramQuotas,
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    await waitFor(() => expect(listProgramQuotas).toHaveBeenCalledTimes(1));
    const firstCall = listProgramQuotas.mock.calls[0] as unknown as [
      { programId: string },
    ];
    expect(firstCall[0].programId).toBe("prog-1");
    expect(firstCall[0].programId).not.toBe("");
  });

  it("never fires listProgramQuotas with an empty programId", async () => {
    const listProgramQuotas = vi.fn(async () => ({ programQuotas: [] }));
    renderWithProviders({
      route: "/admin/academics",
      transport: makeStubTransport([
        CatalogService,
        {
          listPrograms: async () => ({ programs: [mockProgram] }),
          listProgramQuotas,
          listProgramCourses: async () => ({ programCourses: [] }),
          listCourses: async () => ({ courses: [] }),
        },
      ]),
      session: { status: "authenticated", ...adminSession },
      sessionSource: adminSessionSource,
    });

    // Give the page a moment to settle; listProgramQuotas should NOT be called
    // because no Sheet is open yet (programId is never passed as empty string).
    await screen.findByRole("heading", { name: "Académico" });
    await waitFor(() => {
      const callsWithEmptyId = (
        listProgramQuotas.mock.calls as unknown as [{ programId: string }][]
      ).filter((call) => call[0]?.programId === "");
      expect(callsWithEmptyId).toHaveLength(0);
    });
  });

  it("shows Año and Cupo columns (no Carrera column) with correct data", async () => {
    await renderWithSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramQuotas: async () => ({
        programQuotas: [quota2024, quota2025],
      }),
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    await screen.findByText("60");
    expect(screen.getByText("50")).toBeInTheDocument();
    expect(screen.getByText("2024")).toBeInTheDocument();
    expect(screen.getByText("2025")).toBeInTheDocument();

    expect(
      screen.getByRole("columnheader", { name: "Año" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("columnheader", { name: "Cupo" }),
    ).toBeInTheDocument();
    // No "Carrera" column — quotas are scoped to this program
    expect(
      screen.queryByRole("columnheader", { name: "Carrera" }),
    ).not.toBeInTheDocument();
  });

  it("shows loading skeleton while listProgramQuotas is pending", async () => {
    await renderWithSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      // biome-ignore lint/suspicious/noEmptyBlockStatements: intentional never-resolving promise for testing loading state
      listProgramQuotas: () => new Promise(() => {}),
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    const skeleton = await screen.findByRole("status", {
      name: "Cargando cupos",
    });
    expect(skeleton).toBeInTheDocument();
    expect(skeleton).toHaveAttribute("aria-busy", "true");
  });

  it("empty state shows copy and Crear cupo CTA", async () => {
    await renderWithSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramQuotas: async () => ({ programQuotas: [] }),
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    await screen.findByText("Todavía no hay cupos");
    const createButtons = screen.getAllByRole("button", {
      name: /crear cupo/i,
    });
    expect(createButtons.length).toBeGreaterThanOrEqual(1);
  });

  it("transport error shows inline error and retry affordance", async () => {
    await renderWithSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramQuotas: async () => {
        throw new ConnectError("unavailable", Code.Unavailable);
      },
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    await screen.findByText(/No se pudo cargar la lista de cupos/);
    expect(
      screen.getByRole("button", { name: /reintentar/i }),
    ).toBeInTheDocument();
    // No raw error codes in the UI
    expect(screen.queryByText(/Unavailable/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Code\./)).not.toBeInTheDocument();
  });

  it("retry re-fetches and shows rows", async () => {
    const user = userEvent.setup();
    let callCount = 0;
    const listProgramQuotas = vi.fn(async () => {
      callCount++;
      if (callCount === 1) {
        throw new ConnectError("unavailable", Code.Unavailable);
      }
      return { programQuotas: [quota2025] };
    });

    renderWithProviders({
      route: "/admin/academics",
      transport: makeStubTransport([
        CatalogService,
        {
          listPrograms: async () => ({ programs: [mockProgram] }),
          listProgramQuotas,
          listProgramCourses: async () => ({ programCourses: [] }),
          listCourses: async () => ({ courses: [] }),
        },
      ]),
      session: { status: "authenticated", ...adminSession },
      sessionSource: adminSessionSource,
    });

    await screen.findByText("ING-01");
    await user.click(screen.getByRole("button", { name: "Acciones ING-01" }));
    await user.click(screen.getByRole("menuitem", { name: /gestionar/i }));
    await user.click(screen.getByRole("tab", { name: "Cupos" }));

    await screen.findByRole("button", { name: /reintentar/i });
    await user.click(screen.getByRole("button", { name: /reintentar/i }));
    await screen.findByText("60");
  });

  it("shows Editar/Eliminar buttons when session has catalog.manage", async () => {
    await renderWithSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramQuotas: async () => ({ programQuotas: [quota2025] }),
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    await screen.findByText("60");
    expect(
      screen.getByRole("button", { name: `Editar cupo ${quota2025.id}` }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: `Eliminar cupo ${quota2025.id}` }),
    ).toBeInTheDocument();
  });

  it("Crear cupo dialog opens, submits, shows success toast", async () => {
    const createProgramQuota = vi.fn(async () => quota2025);
    const { user } = await renderWithSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramQuotas: async () => ({ programQuotas: [] }),
      createProgramQuota,
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    await screen.findByText("Todavía no hay cupos");

    // Click "Crear cupo" button (the one in the manager header, not empty state)
    const createButtons = screen.getAllByRole("button", {
      name: /crear cupo/i,
    });
    await user.click(createButtons[0]);
    await screen.findByRole("dialog");

    const dialog = screen.getByRole("dialog");
    // No program selector — programId is injected
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
      programId: "prog-1",
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

  it("edit opens dialog pre-filled with year and quota, calls updateProgramQuota", async () => {
    const updateProgramQuota = vi.fn(async () => quota2025);
    const { user } = await renderWithSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramQuotas: async () => ({ programQuotas: [quota2025] }),
      updateProgramQuota,
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    await screen.findByText("60");
    await user.click(
      screen.getByRole("button", { name: `Editar cupo ${quota2025.id}` }),
    );
    await screen.findByRole("dialog");

    const dialog = screen.getByRole("dialog");
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
      { id: string; admissionQuota: number },
    ];
    expect(firstCall[0]).toMatchObject({ id: "q2", admissionQuota: 80 });
    await waitFor(() =>
      expect(toastSuccess).toHaveBeenCalledWith("Cupo actualizado"),
    );
  });

  it("delete opens AlertDialog, confirms, shows success toast", async () => {
    const deleteProgramQuota = vi.fn(async () =>
      create(DeleteProgramQuotaResponseSchema, {}),
    );
    const { user } = await renderWithSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramQuotas: async () => ({ programQuotas: [quota2025] }),
      deleteProgramQuota,
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    await screen.findByText("60");
    await user.click(
      screen.getByRole("button", { name: `Eliminar cupo ${quota2025.id}` }),
    );
    await screen.findByRole("alertdialog");

    await user.click(screen.getByRole("button", { name: "Eliminar" }));

    await waitFor(() =>
      expect(toastSuccess).toHaveBeenCalledWith("Cupo eliminado"),
    );
    await waitFor(() =>
      expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument(),
    );
  });

  it("no Cargar más button — quota list is flat, not paginated", async () => {
    await renderWithSheet({
      listPrograms: async () => ({ programs: [mockProgram] }),
      listProgramQuotas: async () => ({
        programQuotas: [quota2024, quota2025],
      }),
      listProgramCourses: async () => ({ programCourses: [] }),
      listCourses: async () => ({ courses: [] }),
    });

    await screen.findByText("60");
    expect(
      screen.queryByRole("button", { name: /cargar más/i }),
    ).not.toBeInTheDocument();
  });
});
