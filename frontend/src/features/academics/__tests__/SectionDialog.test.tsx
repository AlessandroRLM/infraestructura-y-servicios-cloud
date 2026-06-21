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
  CourseSchema,
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

const mockCourse = create(CourseSchema, {
  id: "course-1",
  code: "CS-101",
  name: "Cálculo",
  credits: 5,
  createdAt: "2024-01-01",
  updatedAt: "2024-01-01",
});

const mockPeriod = create(AcademicPeriodSchema, {
  id: "period-1",
  year: 2024,
  term: 1,
  startDate: "2024-03-01",
  endDate: "2024-07-31",
  createdAt: "2024-01-01",
  updatedAt: "2024-01-01",
});

type CatalogImpl = Partial<ServiceImpl<typeof CatalogService>>;

async function renderSectionsPage(handlers: CatalogImpl = {}) {
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
  await screen.findByRole("heading", { name: "Académico" });
}

describe("SectionDialog — create mode", () => {
  beforeEach(() => {
    toastSuccess.mockClear();
    toastError.mockClear();
  });

  it("success closes dialog, shows success toast, invalidates list", async () => {
    const user = userEvent.setup();
    const createSection = vi.fn(async () => mockSection);
    const listSections = vi.fn(async () => ({ sections: [] }));
    const listCourses = vi.fn(async () => ({ courses: [mockCourse] }));
    const listAcademicPeriods = vi.fn(async () => ({
      academicPeriods: [mockPeriod],
    }));

    await renderSectionsPage({
      createSection,
      listSections,
      listCourses,
      listAcademicPeriods,
    });

    // With empty sections, both header and empty-state buttons render.
    const openButtons = screen.getAllByRole("button", {
      name: /crear sección/i,
    });
    await user.click(openButtons[0]);
    await screen.findByRole("dialog");

    const dialog = screen.getByRole("dialog");

    // Select course
    const courseTrigger = within(dialog).getAllByRole("combobox")[0];
    await user.click(courseTrigger);
    await user.click(await screen.findByRole("option", { name: /Cálculo/i }));

    // Select period
    const periodTrigger = within(dialog).getAllByRole("combobox")[1];
    await user.click(periodTrigger);
    await user.click(
      await screen.findByRole("option", { name: /2024.*Semestre 1/i }),
    );

    // Fill seat capacity
    const capacityInput = within(dialog).getByLabelText("Capacidad");
    await user.clear(capacityInput);
    await user.type(capacityInput, "30");

    const submitBtn = within(dialog).getByRole("button", {
      name: /crear sección/i,
    });
    await user.click(submitBtn);

    await waitFor(() => expect(createSection).toHaveBeenCalledTimes(1));

    await waitFor(() =>
      expect(toastSuccess).toHaveBeenCalledWith("Sección creada"),
    );
    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
    );
  });

  it("transport error shows toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const createSection = vi.fn(async () => {
      throw new ConnectError("internal", Code.Internal);
    });
    const listSections = vi.fn(async () => ({ sections: [] }));
    const listCourses = vi.fn(async () => ({ courses: [mockCourse] }));
    const listAcademicPeriods = vi.fn(async () => ({
      academicPeriods: [mockPeriod],
    }));

    await renderSectionsPage({
      createSection,
      listSections,
      listCourses,
      listAcademicPeriods,
    });

    const openButtons = screen.getAllByRole("button", {
      name: /crear sección/i,
    });
    await user.click(openButtons[0]);
    await screen.findByRole("dialog");

    const dialog = screen.getByRole("dialog");

    const courseTrigger = within(dialog).getAllByRole("combobox")[0];
    await user.click(courseTrigger);
    await user.click(await screen.findByRole("option", { name: /Cálculo/i }));

    const periodTrigger = within(dialog).getAllByRole("combobox")[1];
    await user.click(periodTrigger);
    await user.click(
      await screen.findByRole("option", { name: /2024.*Semestre 1/i }),
    );

    const capacityInput = within(dialog).getByLabelText("Capacidad");
    await user.clear(capacityInput);
    await user.type(capacityInput, "30");

    const submitBtn = within(dialog).getByRole("button", {
      name: /crear sección/i,
    });
    await user.click(submitBtn);

    await waitFor(() => expect(toastError).toHaveBeenCalled());
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });
});

describe("SectionDialog — edit mode", () => {
  beforeEach(() => {
    toastSuccess.mockClear();
    toastError.mockClear();
  });

  it("edit mode pre-fills seat capacity and calls updateSection with id + seatCapacity as number", async () => {
    const user = userEvent.setup();
    const updateSection = vi.fn(async () => mockSection);
    const listSections = vi.fn(async () => ({ sections: [mockSection] }));
    const listCourses = vi.fn(async () => ({ courses: [mockCourse] }));
    const listAcademicPeriods = vi.fn(async () => ({
      academicPeriods: [mockPeriod],
    }));

    await renderSectionsPage({
      updateSection,
      listSections,
      listCourses,
      listAcademicPeriods,
    });

    await screen.findByText("30");

    const editButtons = screen.getAllByRole("button", { name: /editar/i });
    await user.click(editButtons[0]);

    await screen.findByRole("dialog");
    const dialog = screen.getByRole("dialog");

    const capacityInput = within(dialog).getByLabelText(
      "Capacidad",
    ) as HTMLInputElement;
    expect(capacityInput.value).toBe("30");

    await user.clear(capacityInput);
    await user.type(capacityInput, "40");
    await user.click(
      within(dialog).getByRole("button", { name: /guardar cambios/i }),
    );

    await waitFor(() => expect(updateSection).toHaveBeenCalledTimes(1));
    const firstUpdateCall = updateSection.mock.calls[0] as unknown as [
      { id: string; seatCapacity: number },
    ];
    expect(firstUpdateCall[0]).toMatchObject({
      id: "section-1",
      seatCapacity: 40,
    });
    expect(typeof firstUpdateCall[0].seatCapacity).toBe("number");

    await waitFor(() =>
      expect(toastSuccess).toHaveBeenCalledWith("Sección actualizada"),
    );
  });

  it("transport error on update shows toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const updateSection = vi.fn(async () => {
      throw new ConnectError("internal", Code.Internal);
    });
    const listSections = vi.fn(async () => ({ sections: [mockSection] }));
    const listCourses = vi.fn(async () => ({ courses: [mockCourse] }));
    const listAcademicPeriods = vi.fn(async () => ({
      academicPeriods: [mockPeriod],
    }));

    await renderSectionsPage({
      updateSection,
      listSections,
      listCourses,
      listAcademicPeriods,
    });

    await screen.findByText("30");

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
