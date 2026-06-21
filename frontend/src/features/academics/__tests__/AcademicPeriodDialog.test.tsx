import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import {
  AcademicPeriodSchema,
  CatalogService,
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

async function renderPeriodsPage(handlers: CatalogImpl = {}) {
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
  await screen.findByRole("heading", { name: "Académico" });
}

describe("AcademicPeriodDialog — create mode", () => {
  beforeEach(() => {
    toastSuccess.mockClear();
    toastError.mockClear();
  });

  it("success closes dialog, shows success toast, invalidates list", async () => {
    const user = userEvent.setup();
    const createAcademicPeriod = vi.fn(async () => mockPeriod);
    const listAcademicPeriods = vi.fn(async () => ({ academicPeriods: [] }));

    await renderPeriodsPage({ createAcademicPeriod, listAcademicPeriods });

    // With empty periods, both header and empty-state buttons render.
    const openButtons = screen.getAllByRole("button", {
      name: /crear período/i,
    });
    await user.click(openButtons[0]);
    await screen.findByRole("dialog");

    const dialog = screen.getByRole("dialog");
    const yearInput = within(dialog).getByLabelText("Año");
    const termInput = within(dialog).getByLabelText("Semestre");
    const startInput = within(dialog).getByLabelText("Inicio");
    const endInput = within(dialog).getByLabelText("Término");

    await user.clear(yearInput);
    await user.type(yearInput, "2026");
    await user.clear(termInput);
    await user.type(termInput, "1");
    await user.type(startInput, "2026-03-01");
    await user.type(endInput, "2026-07-15");

    const submitBtn = within(dialog).getByRole("button", {
      name: /crear período/i,
    });
    await user.click(submitBtn);

    await waitFor(() => expect(createAcademicPeriod).toHaveBeenCalledTimes(1));
    const firstCall = createAcademicPeriod.mock.calls[0] as unknown as [
      { year: number; term: number; startDate: string; endDate: string },
    ];
    expect(firstCall[0].year).toBe(2026);
    expect(typeof firstCall[0].year).toBe("number");
    expect(firstCall[0].term).toBe(1);

    await waitFor(() =>
      expect(toastSuccess).toHaveBeenCalledWith("Período creado"),
    );
    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
    );
  });

  it("AlreadyExists shows inline year error, no toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const createAcademicPeriod = vi.fn(async () => {
      throw new ConnectError("duplicate", Code.AlreadyExists);
    });
    const listAcademicPeriods = vi.fn(async () => ({ academicPeriods: [] }));

    await renderPeriodsPage({ createAcademicPeriod, listAcademicPeriods });

    const openButtons = screen.getAllByRole("button", {
      name: /crear período/i,
    });
    await user.click(openButtons[0]);
    await screen.findByRole("dialog");

    const dialog = screen.getByRole("dialog");
    await user.clear(within(dialog).getByLabelText("Año"));
    await user.type(within(dialog).getByLabelText("Año"), "2025");
    await user.clear(within(dialog).getByLabelText("Semestre"));
    await user.type(within(dialog).getByLabelText("Semestre"), "1");
    await user.type(within(dialog).getByLabelText("Inicio"), "2025-03-01");
    await user.type(within(dialog).getByLabelText("Término"), "2025-07-15");

    await user.click(
      within(dialog).getByRole("button", { name: /crear período/i }),
    );

    await waitFor(() =>
      expect(
        screen.getByText("Ya existe un período con ese año y semestre"),
      ).toBeInTheDocument(),
    );
    expect(toastError).not.toHaveBeenCalled();
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("transport error shows toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const createAcademicPeriod = vi.fn(async () => {
      throw new ConnectError("internal", Code.Internal);
    });
    const listAcademicPeriods = vi.fn(async () => ({ academicPeriods: [] }));

    await renderPeriodsPage({ createAcademicPeriod, listAcademicPeriods });

    const openButtons = screen.getAllByRole("button", {
      name: /crear período/i,
    });
    await user.click(openButtons[0]);
    await screen.findByRole("dialog");

    const dialog = screen.getByRole("dialog");
    await user.clear(within(dialog).getByLabelText("Año"));
    await user.type(within(dialog).getByLabelText("Año"), "2026");
    await user.clear(within(dialog).getByLabelText("Semestre"));
    await user.type(within(dialog).getByLabelText("Semestre"), "2");
    await user.type(within(dialog).getByLabelText("Inicio"), "2026-08-01");
    await user.type(within(dialog).getByLabelText("Término"), "2026-12-15");

    await user.click(
      within(dialog).getByRole("button", { name: /crear período/i }),
    );

    await waitFor(() => expect(toastError).toHaveBeenCalled());
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });
});

describe("AcademicPeriodDialog — edit mode", () => {
  beforeEach(() => {
    toastSuccess.mockClear();
    toastError.mockClear();
  });

  it("edit mode pre-fills all fields and calls updateAcademicPeriod with id + numeric year/term", async () => {
    const user = userEvent.setup();
    const updateAcademicPeriod = vi.fn(async () => mockPeriod);
    const listAcademicPeriods = vi.fn(async () => ({
      academicPeriods: [mockPeriod],
    }));

    await renderPeriodsPage({ updateAcademicPeriod, listAcademicPeriods });

    await screen.findByText("2025-03-01");

    const editButton = screen.getByRole("button", {
      name: /editar período 2025 · Semestre 1/i,
    });
    await user.click(editButton);

    await screen.findByRole("dialog");
    const dialog = screen.getByRole("dialog");

    const yearInput = within(dialog).getByLabelText("Año") as HTMLInputElement;
    const termInput = within(dialog).getByLabelText(
      "Semestre",
    ) as HTMLInputElement;
    expect(yearInput.value).toBe("2025");
    expect(termInput.value).toBe("1");

    await user.click(
      within(dialog).getByRole("button", { name: /guardar cambios/i }),
    );

    await waitFor(() => expect(updateAcademicPeriod).toHaveBeenCalledTimes(1));
    const firstCall = updateAcademicPeriod.mock.calls[0] as unknown as [
      {
        id: string;
        year: number;
        term: number;
        startDate: string;
        endDate: string;
      },
    ];
    expect(firstCall[0]).toMatchObject({
      id: "period-1",
      year: 2025,
      term: 1,
    });
    expect(typeof firstCall[0].year).toBe("number");

    await waitFor(() =>
      expect(toastSuccess).toHaveBeenCalledWith("Período actualizado"),
    );
  });

  it("transport error on update shows toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const updateAcademicPeriod = vi.fn(async () => {
      throw new ConnectError("internal", Code.Internal);
    });
    const listAcademicPeriods = vi.fn(async () => ({
      academicPeriods: [mockPeriod],
    }));

    await renderPeriodsPage({ updateAcademicPeriod, listAcademicPeriods });
    await screen.findByText("2025-03-01");

    await user.click(
      screen.getByRole("button", { name: /editar período 2025 · Semestre 1/i }),
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
