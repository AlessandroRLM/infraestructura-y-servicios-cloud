/**
 * CreateEnrollmentDialog component tests.
 *
 * Covers:
 *  - Submit is disabled when no fields are set.
 *  - Submit remains disabled with program+year but no student.
 *  - Happy path: calls createEnrollment with correct args + success toast + close.
 *  - AlreadyExists domain error → inline message (no raw code, no toast).
 *  - Transport error → toast.
 *  - Year untypeable-fix: partial/out-of-range year keeps submit disabled.
 */
import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import {
  CatalogService,
  ListProgramsResponseSchema,
  ProgramSchema,
} from "@/gen/catalog/v1/catalog_pb";
import {
  EnrollmentSchema,
  EnrollmentService,
} from "@/gen/enrollment/v1/enrollment_pb";
import {
  IamService,
  ListUsersResponseSchema,
  UserSummarySchema,
} from "@/gen/iam/v1/iam_pb";
import { renderComponent } from "@/test";
import { CreateEnrollmentDialog } from "../components/CreateEnrollmentDialog";

const { toastSuccess, toastError } = vi.hoisted(() => ({
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));
vi.mock("sonner", () => ({
  toast: { success: toastSuccess, error: toastError },
}));

function makeProgram(id: string, code: string, name: string) {
  return create(ProgramSchema, {
    id,
    code,
    name,
    createdAt: "",
    updatedAt: "",
  });
}

function makeUser(id: string, email: string, displayName: string) {
  return create(UserSummarySchema, {
    id,
    email,
    displayName,
    roles: [],
    status: 1, // ACTIVE
  });
}

type EnrollmentImpl = Partial<ServiceImpl<typeof EnrollmentService>>;
type CatalogImpl = Partial<ServiceImpl<typeof CatalogService>>;
type IamImpl = Partial<ServiceImpl<typeof IamService>>;

const mockStudent = makeUser("student-1", "ana@test.com", "Ana García");
const mockProgram = makeProgram("prog-1", "ICI", "Ingeniería Civil");

function makeDefaultTransport(enrollmentImpl: EnrollmentImpl = {}) {
  const catalogImpl: CatalogImpl = {
    listPrograms: async () =>
      create(ListProgramsResponseSchema, {
        programs: [mockProgram],
        nextPageToken: "",
      }),
  };
  const iamImpl: IamImpl = {
    listUsers: async () =>
      create(ListUsersResponseSchema, {
        users: [mockStudent],
        nextPageToken: "",
      }),
  };

  return makeStubTransport(
    [EnrollmentService, enrollmentImpl],
    [CatalogService, catalogImpl],
    [IamService, iamImpl],
  );
}

/**
 * Selects a student from StudentPicker. Waits for IamService to return users.
 */
async function selectStudent(user: ReturnType<typeof userEvent.setup>) {
  const studentTrigger = screen.getByRole("combobox", {
    name: /seleccionar estudiante/i,
  });
  await user.click(studentTrigger);
  await waitFor(() =>
    expect(screen.getByText(/ana@test\.com/i)).toBeInTheDocument(),
  );
  await user.click(screen.getByText(/ana@test\.com/i));
}

/**
 * Selects a program from ProgramPicker. Waits for CatalogService to return programs.
 */
async function selectProgram(user: ReturnType<typeof userEvent.setup>) {
  const programTrigger = screen.getByRole("combobox", {
    name: /seleccionar programa/i,
  });
  await user.click(programTrigger);
  await waitFor(() =>
    expect(screen.getByText("ICI — Ingeniería Civil")).toBeInTheDocument(),
  );
  await user.click(screen.getByText("ICI — Ingeniería Civil"));
}

async function fillYear(
  user: ReturnType<typeof userEvent.setup>,
  yearStr: string,
) {
  const yearInput = screen.getByPlaceholderText("Ej. 2026");
  await user.type(yearInput, yearStr);
}

describe("CreateEnrollmentDialog — submit disabled state", () => {
  it("submit is disabled when no fields are set", () => {
    renderComponent(
      <CreateEnrollmentDialog open onOpenChange={vi.fn()} />,
      { transport: makeDefaultTransport() },
    );

    const submitBtn = screen.getByRole("button", { name: /crear matrícula/i });
    expect(submitBtn).toBeDisabled();
  });

  it("submit remains disabled with program+year but no student", async () => {
    const user = userEvent.setup();
    renderComponent(
      <CreateEnrollmentDialog open onOpenChange={vi.fn()} />,
      { transport: makeDefaultTransport() },
    );

    await selectProgram(user);
    await fillYear(user, "2026");

    const submitBtn = screen.getByRole("button", { name: /crear matrícula/i });
    expect(submitBtn).toBeDisabled();
  });

  it("submit remains disabled with out-of-range year", async () => {
    const user = userEvent.setup();
    renderComponent(
      <CreateEnrollmentDialog open onOpenChange={vi.fn()} />,
      { transport: makeDefaultTransport() },
    );

    await selectStudent(user);
    await selectProgram(user);
    await fillYear(user, "2"); // out-of-range

    const submitBtn = screen.getByRole("button", { name: /crear matrícula/i });
    expect(submitBtn).toBeDisabled();
  });
});

describe("CreateEnrollmentDialog — happy path", () => {
  it("calls createEnrollment with correct args, shows success toast, and closes", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    const mockEnrollment = create(EnrollmentSchema, {
      id: "enroll-1",
      studentId: "student-1",
      programId: "prog-1",
      year: 2026,
      status: "pending",
      createdAt: "",
      updatedAt: "",
      programName: "Ingeniería Civil",
      studentName: "Ana García",
    });
    const createEnrollment = vi.fn(async () => mockEnrollment);

    renderComponent(
      <CreateEnrollmentDialog open onOpenChange={onOpenChange} />,
      { transport: makeDefaultTransport({ createEnrollment }) },
    );

    await selectStudent(user);
    await selectProgram(user);
    await fillYear(user, "2026");

    const submitBtn = await waitFor(() => {
      const btn = screen.getByRole("button", { name: /crear matrícula/i });
      expect(btn).not.toBeDisabled();
      return btn;
    });

    await user.click(submitBtn);

    await waitFor(() => {
      expect(createEnrollment).toHaveBeenCalledWith(
        expect.objectContaining({
          studentId: "student-1",
          programId: "prog-1",
          year: 2026,
        }),
        expect.anything(),
      );
    });

    await waitFor(() =>
      expect(toastSuccess).toHaveBeenCalledWith("Matrícula creada"),
    );
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
  });
});

describe("CreateEnrollmentDialog — domain error (AlreadyExists)", () => {
  it("shows inline domain message, no toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    const createEnrollment = vi.fn(async () => {
      throw new ConnectError("already exists", Code.AlreadyExists);
    });

    renderComponent(
      <CreateEnrollmentDialog open onOpenChange={onOpenChange} />,
      { transport: makeDefaultTransport({ createEnrollment }) },
    );

    await selectStudent(user);
    await selectProgram(user);
    await fillYear(user, "2026");

    const submitBtn = await waitFor(() => {
      const btn = screen.getByRole("button", { name: /crear matrícula/i });
      expect(btn).not.toBeDisabled();
      return btn;
    });
    await user.click(submitBtn);

    await waitFor(() =>
      expect(
        screen.getByText(/ya existe una matrícula/i),
      ).toBeInTheDocument(),
    );

    // No raw error code visible
    expect(screen.queryByText(/AlreadyExists/)).not.toBeInTheDocument();
    expect(screen.queryByText(/ALREADY_EXISTS/)).not.toBeInTheDocument();
    // No toast
    expect(toastError).not.toHaveBeenCalled();
    // Dialog stays open
    expect(onOpenChange).not.toHaveBeenCalledWith(false);
  });
});

describe("CreateEnrollmentDialog — transport error", () => {
  it("shows toast.error, dialog stays open, no raw code", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    const createEnrollment = vi.fn(async () => {
      throw new ConnectError("internal", Code.Internal);
    });

    renderComponent(
      <CreateEnrollmentDialog open onOpenChange={onOpenChange} />,
      { transport: makeDefaultTransport({ createEnrollment }) },
    );

    await selectStudent(user);
    await selectProgram(user);
    await fillYear(user, "2026");

    const submitBtn = await waitFor(() => {
      const btn = screen.getByRole("button", { name: /crear matrícula/i });
      expect(btn).not.toBeDisabled();
      return btn;
    });
    await user.click(submitBtn);

    await waitFor(() => expect(toastError).toHaveBeenCalled());

    expect(screen.queryByText(/Internal/)).not.toBeInTheDocument();
    expect(screen.queryByText(/INTERNAL/)).not.toBeInTheDocument();
    expect(onOpenChange).not.toHaveBeenCalledWith(false);
  });
});
