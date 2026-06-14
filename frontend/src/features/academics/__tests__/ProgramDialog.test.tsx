import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import { CatalogService, ProgramSchema } from "@/gen/catalog/v1/catalog_pb";
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

async function renderAcademicsPage(handlers: CatalogImpl = {}) {
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
  await screen.findByRole("heading", { name: "Académico" });
}

describe("ProgramDialog — create mode", () => {
  it("S-05: success closes dialog, shows success toast, invalidates list", async () => {
    const user = userEvent.setup();
    const createProgram = vi.fn(async () => mockProgram);
    const listPrograms = vi.fn(async () => ({ programs: [] }));

    await renderAcademicsPage({ createProgram, listPrograms });

    // Use getAllByRole — when list is empty, both the header and empty-state render the button.
    const openButtons = screen.getAllByRole("button", {
      name: /crear carrera/i,
    });
    await user.click(openButtons[0]);
    await screen.findByRole("dialog");

    const dialog = screen.getByRole("dialog");
    await user.type(within(dialog).getByLabelText("Código"), "ING-01");
    await user.type(
      within(dialog).getByLabelText("Nombre"),
      "Ingeniería de Software",
    );

    const submitBtn = within(dialog).getByRole("button", {
      name: /crear carrera/i,
    });
    await user.click(submitBtn);

    await waitFor(() => expect(createProgram).toHaveBeenCalledTimes(1));
    await waitFor(() =>
      expect(toastSuccess).toHaveBeenCalledWith("Carrera creada"),
    );
    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
    );
  });

  it("S-07: AlreadyExists shows inline code error, no toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const createProgram = vi.fn(async () => {
      throw new ConnectError("duplicate", Code.AlreadyExists);
    });
    const listPrograms = vi.fn(async () => ({ programs: [] }));

    await renderAcademicsPage({ createProgram, listPrograms });

    const openButtons = screen.getAllByRole("button", {
      name: /crear carrera/i,
    });
    await user.click(openButtons[0]);
    await screen.findByRole("dialog");

    const dialog = screen.getByRole("dialog");
    await user.type(within(dialog).getByLabelText("Código"), "DUP-01");
    await user.type(within(dialog).getByLabelText("Nombre"), "Duplicado");

    const submitBtn = within(dialog).getByRole("button", {
      name: /crear carrera/i,
    });
    await user.click(submitBtn);

    await waitFor(() =>
      expect(
        screen.getByText("Ya existe una carrera con ese código"),
      ).toBeInTheDocument(),
    );
    expect(toastError).not.toHaveBeenCalled();
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("S-08: transport error shows toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const createProgram = vi.fn(async () => {
      throw new ConnectError("internal", Code.Internal);
    });
    const listPrograms = vi.fn(async () => ({ programs: [] }));

    await renderAcademicsPage({ createProgram, listPrograms });

    const openButtons = screen.getAllByRole("button", {
      name: /crear carrera/i,
    });
    await user.click(openButtons[0]);
    await screen.findByRole("dialog");

    const dialog = screen.getByRole("dialog");
    await user.type(within(dialog).getByLabelText("Código"), "OK-01");
    await user.type(within(dialog).getByLabelText("Nombre"), "Nombre válido");

    const submitBtn = within(dialog).getByRole("button", {
      name: /crear carrera/i,
    });
    await user.click(submitBtn);

    await waitFor(() => expect(toastError).toHaveBeenCalled());
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });
});

describe("ProgramDialog — edit mode error paths (S-11)", () => {
  beforeEach(() => {
    toastSuccess.mockClear();
    toastError.mockClear();
  });

  it("S-11a: AlreadyExists on update shows inline code error, no toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const updateProgram = vi.fn(async () => {
      throw new ConnectError("duplicate", Code.AlreadyExists);
    });
    const listPrograms = vi.fn(async () => ({ programs: [mockProgram] }));

    await renderAcademicsPage({ updateProgram, listPrograms });
    await screen.findByText("ING-01");

    const editButtons = screen.getAllByRole("button", { name: /editar/i });
    await user.click(editButtons[0]);
    await screen.findByRole("dialog");

    const dialog = screen.getByRole("dialog");
    await user.click(
      within(dialog).getByRole("button", { name: /guardar cambios/i }),
    );

    await waitFor(() =>
      expect(
        screen.getByText("Ya existe una carrera con ese código"),
      ).toBeInTheDocument(),
    );
    expect(toastError).not.toHaveBeenCalled();
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("S-11b: transport error on update shows toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const updateProgram = vi.fn(async () => {
      throw new ConnectError("internal", Code.Internal);
    });
    const listPrograms = vi.fn(async () => ({ programs: [mockProgram] }));

    await renderAcademicsPage({ updateProgram, listPrograms });
    await screen.findByText("ING-01");

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

describe("ProgramDialog — edit mode", () => {
  it("S-09/S-10: edit mode pre-fills fields and calls updateProgram with id", async () => {
    const user = userEvent.setup();
    const updateProgram = vi.fn(async () => mockProgram);
    const listPrograms = vi.fn(async () => ({ programs: [mockProgram] }));

    await renderAcademicsPage({ updateProgram, listPrograms });

    await screen.findByText("ING-01");

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
    expect(codeInput.value).toBe("ING-01");
    expect(nameInput.value).toBe("Ingeniería de Software");

    await user.clear(nameInput);
    await user.type(nameInput, "Ingeniería de Sistemas");
    await user.click(
      within(dialog).getByRole("button", { name: /guardar cambios/i }),
    );

    await waitFor(() => expect(updateProgram).toHaveBeenCalledTimes(1));
    const firstCall = updateProgram.mock.calls[0] as unknown as [
      { id: string; code: string; name: string },
    ];
    expect(firstCall[0]).toMatchObject({
      id: "prog-1",
      code: "ING-01",
      name: "Ingeniería de Sistemas",
    });
    await waitFor(() =>
      expect(toastSuccess).toHaveBeenCalledWith("Carrera actualizada"),
    );
  });
});
