import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import {
  CatalogService,
  RemoveTeacherFromSectionResponseSchema,
  SectionSchema,
  SectionTeacherSchema,
} from "@/gen/catalog/v1/catalog_pb";
import { IamService, UserSummarySchema } from "@/gen/iam/v1/iam_pb";
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

const teacher1 = create(UserSummarySchema, {
  id: "teacher-1",
  email: "teacher1@test.com",
  displayName: "Prof. Ana García",
  roles: ["teacher"],
});

const teacher2 = create(UserSummarySchema, {
  id: "teacher-2",
  email: "teacher2@test.com",
  displayName: "Prof. Luis Pérez",
  roles: ["teacher"],
});

const sectionTeacher1 = create(SectionTeacherSchema, {
  sectionId: "section-1",
  teacherId: "teacher-1",
  createdAt: "2024-01-01",
});

type CatalogImpl = Partial<ServiceImpl<typeof CatalogService>>;
type IamImpl = Partial<ServiceImpl<typeof IamService>>;

const adminSession = {
  status: "authenticated" as const,
  userId: "1",
  email: "admin@test.com",
  roles: ["admin"],
  permissions: ["catalog.manage"],
};

function renderWithBothServices(
  catalogHandlers: CatalogImpl,
  iamHandlers: IamImpl,
) {
  return renderWithProviders({
    route: "/admin/academics?tab=sections",
    transport: makeStubTransport(
      [CatalogService, catalogHandlers],
      [IamService, iamHandlers],
    ),
    session: adminSession,
  });
}

describe("SectionTeachersManager (teacher M:N)", () => {
  beforeEach(() => {
    toastSuccess.mockClear();
    toastError.mockClear();
  });

  it("opens teacher manager sheet for a section row and shows assigned teachers", async () => {
    const user = userEvent.setup();

    renderWithBothServices(
      {
        listSections: async () => ({ sections: [mockSection] }),
        listSectionTeachers: async () => ({
          sectionTeachers: [sectionTeacher1],
        }),
      },
      {
        listUsers: async () => ({
          users: [teacher1, teacher2],
          nextPageToken: "",
        }),
      },
    );

    await screen.findByText("30");

    // Click the manage teachers button
    const manageBtn = screen.getByRole("button", {
      name: /docentes/i,
    });
    await user.click(manageBtn);

    // Sheet should open and show the teacher
    await screen.findByText("Prof. Ana García");
  });

  it("assigns a new teacher successfully", async () => {
    const user = userEvent.setup();
    const assignTeacherToSection = vi.fn(async () => sectionTeacher1);

    renderWithBothServices(
      {
        listSections: async () => ({ sections: [mockSection] }),
        listSectionTeachers: async () => ({ sectionTeachers: [] }),
        assignTeacherToSection,
      },
      {
        listUsers: async () => ({
          users: [teacher1, teacher2],
          nextPageToken: "",
        }),
      },
    );

    await screen.findByText("30");

    const manageBtn = screen.getByRole("button", { name: /docentes/i });
    await user.click(manageBtn);

    await screen.findByRole("combobox", { name: /agregar docente/i });

    await user.click(
      screen.getByRole("combobox", { name: /agregar docente/i }),
    );
    await user.click(
      await screen.findByRole("option", { name: /Ana García/i }),
    );

    await waitFor(() =>
      expect(assignTeacherToSection).toHaveBeenCalledTimes(1),
    );
    await waitFor(() =>
      expect(toastSuccess).toHaveBeenCalledWith("Docente asignado."),
    );
  });

  it("shows error toast when assign fails with AlreadyExists", async () => {
    const user = userEvent.setup();
    const assignTeacherToSection = vi.fn(async () => {
      throw new ConnectError("already assigned", Code.AlreadyExists);
    });

    renderWithBothServices(
      {
        listSections: async () => ({ sections: [mockSection] }),
        listSectionTeachers: async () => ({ sectionTeachers: [] }),
        assignTeacherToSection,
      },
      {
        listUsers: async () => ({
          users: [teacher1],
          nextPageToken: "",
        }),
      },
    );

    await screen.findByText("30");
    const manageBtn = screen.getByRole("button", { name: /docentes/i });
    await user.click(manageBtn);

    await screen.findByRole("combobox", { name: /agregar docente/i });
    await user.click(
      screen.getByRole("combobox", { name: /agregar docente/i }),
    );
    await user.click(
      await screen.findByRole("option", { name: /Ana García/i }),
    );

    await waitFor(() =>
      expect(toastError).toHaveBeenCalledWith(
        "Este docente ya está asignado a la sección.",
      ),
    );
  });

  it("removes a teacher after confirmation", async () => {
    const user = userEvent.setup();
    const removeTeacherFromSection = vi.fn(async () =>
      create(RemoveTeacherFromSectionResponseSchema, {}),
    );

    renderWithBothServices(
      {
        listSections: async () => ({ sections: [mockSection] }),
        listSectionTeachers: async () => ({
          sectionTeachers: [sectionTeacher1],
        }),
        removeTeacherFromSection,
      },
      {
        listUsers: async () => ({
          users: [teacher1, teacher2],
          nextPageToken: "",
        }),
      },
    );

    await screen.findByText("30");

    const manageBtn = screen.getByRole("button", { name: /docentes/i });
    await user.click(manageBtn);

    await screen.findByText("Prof. Ana García");

    // Click remove button for teacher1
    const removeBtn = screen.getByRole("button", {
      name: /quitar/i,
    });
    await user.click(removeBtn);

    // Confirmation dialog appears
    await screen.findByRole("alertdialog");
    await user.click(screen.getByRole("button", { name: "Quitar" }));

    await waitFor(() =>
      expect(removeTeacherFromSection).toHaveBeenCalledTimes(1),
    );
    await waitFor(() =>
      expect(toastSuccess).toHaveBeenCalledWith("Docente quitado."),
    );
  });
});
