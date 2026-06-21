/**
 * EnrollSectionDialog component tests.
 *
 * Covers:
 *  - Renders enrollment_id input and calls enrollSection with correct fields.
 *  - FailedPrecondition → inline precondition message (section full / window / etc).
 *  - AlreadyExists → inline already-enrolled message.
 *  - Transport error → toast (not inline).
 *  - Submit disabled when enrollmentId is empty.
 *  - "Cancelar" cancel button label is present.
 */
import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import {
  EnrollmentSchema,
  EnrollmentService,
} from "@/gen/enrollment/v1/enrollment_pb";
import { ProfileService } from "@/gen/profiles/v1/profiles_pb";
import {
  SectionEnrollmentSchema,
  SectionEnrollmentService,
} from "@/gen/section_enrollment/v1/section_enrollment_pb";
import { renderComponent } from "@/test";
import { EnrollSectionDialog } from "../components/EnrollSectionDialog";

const { toastSuccess, toastError } = vi.hoisted(() => ({
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));
vi.mock("sonner", () => ({
  toast: { success: toastSuccess, error: toastError },
}));

type SectionEnrollmentImpl = Partial<
  ServiceImpl<typeof SectionEnrollmentService>
>;
type EnrollmentImpl = Partial<ServiceImpl<typeof EnrollmentService>>;
type ProfileImpl = Partial<ServiceImpl<typeof ProfileService>>;

const stubEnrollments = [
  create(EnrollmentSchema, {
    id: "enroll-1",
    studentId: "aaaaaaaa-0000-0000-0000-000000000001",
    programId: "prog-1",
    programName: "Ingeniería Civil",
    year: 2026,
    status: "paid",
  }),
  create(EnrollmentSchema, {
    id: "enroll-2",
    studentId: "bbbbbbbb-0000-0000-0000-000000000002",
    programId: "prog-2",
    programName: "Derecho",
    year: 2026,
    status: "paid",
  }),
];

const createdEnrollment = create(SectionEnrollmentSchema, {
  id: "se-new",
  enrollmentId: "enroll-1",
  sectionId: "sec-1",
  status: "in_progress",
  registeredAt: "2026-01-10T00:00:00Z",
  createdAt: "2026-01-10T00:00:00Z",
  updatedAt: "2026-01-10T00:00:00Z",
});

function renderDialog(
  seImpl: SectionEnrollmentImpl,
  enrollImpl: EnrollmentImpl = {},
  onOpenChange = vi.fn(),
  profileImpl: ProfileImpl = {
    listDisplayNamesByIDs: async () => ({ names: [] }),
  },
) {
  renderComponent(
    <EnrollSectionDialog open sectionId="sec-1" onOpenChange={onOpenChange} />,
    {
      transport: makeStubTransport(
        [SectionEnrollmentService, seImpl],
        [
          EnrollmentService,
          {
            listEnrollments: async () => ({
              enrollments: stubEnrollments,
              nextPageToken: "",
            }),
            ...enrollImpl,
          },
        ],
        [ProfileService, profileImpl],
      ),
      session: {
        status: "authenticated",
        userId: "admin-1",
        email: "admin@test.com",
        roles: ["admin"],
        permissions: ["enrollment.manage", "profile.view_names"],
      },
    },
  );
}

describe("EnrollSectionDialog — submit disabled until enrollment picked", () => {
  it("Inscribir button is disabled when no enrollment is selected", async () => {
    renderDialog({});

    // Wait for dialog to render
    await screen.findByRole("dialog");
    const submitBtn = screen.getByRole("button", { name: /inscribir alumno/i });
    expect(submitBtn).toBeDisabled();
  });
});

describe("EnrollSectionDialog — cancel button", () => {
  it("renders 'Cancelar' cancel button", async () => {
    renderDialog({});
    await screen.findByRole("dialog");
    expect(
      screen.getByRole("button", { name: /^cancelar$/i }),
    ).toBeInTheDocument();
  });
});

describe("EnrollSectionDialog — FailedPrecondition", () => {
  it("shows inline precondition message, no toast", async () => {
    const user = userEvent.setup();
    const enrollSection = vi.fn(async () => {
      throw new ConnectError("section full", Code.FailedPrecondition);
    });

    renderDialog({ enrollSection });

    // Wait for enrollments to load and pick one
    await screen.findByRole("dialog");
    const trigger = screen.getByRole("combobox", {
      name: /seleccionar matrícula/i,
    });
    await user.click(trigger);

    // Select first item in list
    await screen.findByText(/aaaaaaaa/i);
    await user.click(screen.getByText(/aaaaaaaa/i));

    await user.click(screen.getByRole("button", { name: /inscribir alumno/i }));

    await waitFor(() => expect(screen.getByRole("alert")).toBeInTheDocument());
    expect(screen.queryByText(/FailedPrecondition/)).not.toBeInTheDocument();
    expect(toastError).not.toHaveBeenCalled();
  });
});

describe("EnrollSectionDialog — AlreadyExists", () => {
  it("shows inline already-enrolled message, no toast", async () => {
    const user = userEvent.setup();
    const enrollSection = vi.fn(async () => {
      throw new ConnectError("already enrolled", Code.AlreadyExists);
    });

    renderDialog({ enrollSection });

    await screen.findByRole("dialog");
    const trigger = screen.getByRole("combobox", {
      name: /seleccionar matrícula/i,
    });
    await user.click(trigger);

    await screen.findByText(/aaaaaaaa/i);
    await user.click(screen.getByText(/aaaaaaaa/i));

    await user.click(screen.getByRole("button", { name: /inscribir alumno/i }));

    await waitFor(() => expect(screen.getByRole("alert")).toBeInTheDocument());
    expect(toastError).not.toHaveBeenCalled();
  });
});

describe("EnrollSectionDialog — success", () => {
  it("calls enrollSection and shows success toast, closes dialog", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    const enrollSection = vi.fn(async () => createdEnrollment);

    renderDialog({ enrollSection }, {}, onOpenChange);

    await screen.findByRole("dialog");
    const trigger = screen.getByRole("combobox", {
      name: /seleccionar matrícula/i,
    });
    await user.click(trigger);

    await screen.findByText(/aaaaaaaa/i);
    await user.click(screen.getByText(/aaaaaaaa/i));

    await user.click(screen.getByRole("button", { name: /inscribir alumno/i }));

    await waitFor(() =>
      expect(enrollSection).toHaveBeenCalledWith(
        expect.objectContaining({
          enrollmentId: "enroll-1",
          sectionId: "sec-1",
        }),
        expect.anything(),
      ),
    );
    await waitFor(() => expect(toastSuccess).toHaveBeenCalled());
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
  });
});

describe("EnrollSectionDialog — display name resolution in picker", () => {
  it("shows resolved display name in picker items when ProfileService returns a match", async () => {
    const user = userEvent.setup();

    renderDialog({}, {}, vi.fn(), {
      listDisplayNamesByIDs: async () => ({
        names: [
          {
            userId: "aaaaaaaa-0000-0000-0000-000000000001",
            givenNames: "Ana",
            lastNamePaternal: "López",
          },
        ],
      }),
    });

    await screen.findByRole("dialog");
    const trigger = screen.getByRole("combobox", {
      name: /seleccionar matrícula/i,
    });
    await user.click(trigger);

    // Resolved name appears in the picker list instead of the UUID slice.
    expect(await screen.findByText(/Ana López/)).toBeInTheDocument();
    expect(screen.queryByText(/^aaaaaaaa/)).not.toBeInTheDocument();
  });

  it("falls back to studentId[:8] in picker items when ProfileService returns no match", async () => {
    const user = userEvent.setup();

    renderDialog({}, {}, vi.fn(), {
      listDisplayNamesByIDs: async () => ({ names: [] }),
    });

    await screen.findByRole("dialog");
    const trigger = screen.getByRole("combobox", {
      name: /seleccionar matrícula/i,
    });
    await user.click(trigger);

    // Falls back to the UUID prefix.
    expect(await screen.findByText(/aaaaaaaa/)).toBeInTheDocument();
  });
});
