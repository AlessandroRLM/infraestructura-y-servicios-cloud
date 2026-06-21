/**
 * WithdrawSectionDialog component tests.
 *
 * Covers:
 *  - Confirm calls withdrawSection({id}) and shows success toast.
 *  - FailedPrecondition → inline precondition message (no raw code, no toast).
 *  - Transport error → inline transport message (no toast).
 *  - "Volver" cancel button label is present.
 *  - No auto-close on action click (e.preventDefault pattern).
 */
import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import { ProfileService } from "@/gen/profiles/v1/profiles_pb";
import {
  SectionEnrollmentSchema,
  SectionEnrollmentService,
  WithdrawSectionResponseSchema,
} from "@/gen/section_enrollment/v1/section_enrollment_pb";
import { renderComponent } from "@/test";
import { WithdrawSectionDialog } from "../components/WithdrawSectionDialog";

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
type ProfileImpl = Partial<ServiceImpl<typeof ProfileService>>;

const mockEnrollment = create(SectionEnrollmentSchema, {
  id: "se-1",
  enrollmentId: "enroll-1",
  sectionId: "sec-1",
  status: "in_progress",
  registeredAt: "2026-01-10T00:00:00Z",
  createdAt: "2026-01-10T00:00:00Z",
  updatedAt: "2026-01-10T00:00:00Z",
  studentId: "student-aaaaaaaa-0000-0000-0000-000000000001",
});

function renderDialog(
  impl: SectionEnrollmentImpl,
  onOpenChange = vi.fn(),
  profileImpl: ProfileImpl = {
    listDisplayNamesByIDs: async () => ({ names: [] }),
  },
) {
  renderComponent(
    <WithdrawSectionDialog
      open
      onOpenChange={onOpenChange}
      sectionEnrollment={mockEnrollment}
    />,
    {
      transport: makeStubTransport(
        [SectionEnrollmentService, impl],
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

describe("WithdrawSectionDialog — confirm success", () => {
  it("calls withdrawSection({id}) and shows success toast", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    const withdrawSection = vi.fn(async () =>
      create(WithdrawSectionResponseSchema, {}),
    );

    renderDialog({ withdrawSection }, onOpenChange);

    await user.click(screen.getByRole("button", { name: /retirar/i }));

    await waitFor(() =>
      expect(withdrawSection).toHaveBeenCalledWith(
        expect.objectContaining({ id: "se-1" }),
        expect.anything(),
      ),
    );
    await waitFor(() => expect(toastSuccess).toHaveBeenCalled());
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
  });

  it("cancel button label is 'Volver'", () => {
    renderDialog({});
    expect(screen.getByRole("button", { name: "Volver" })).toBeInTheDocument();
  });
});

describe("WithdrawSectionDialog — FailedPrecondition", () => {
  it("shows inline precondition message, no toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    const withdrawSection = vi.fn(async () => {
      throw new ConnectError("not in_progress", Code.FailedPrecondition);
    });

    renderDialog({ withdrawSection }, onOpenChange);

    await user.click(screen.getByRole("button", { name: /retirar/i }));

    await waitFor(() => expect(screen.getByRole("alert")).toBeInTheDocument());

    expect(screen.queryByText(/FailedPrecondition/)).not.toBeInTheDocument();
    expect(screen.queryByText(/FAILED_PRECONDITION/)).not.toBeInTheDocument();
    expect(toastError).not.toHaveBeenCalled();
    expect(onOpenChange).not.toHaveBeenCalledWith(false);
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
  });
});

describe("WithdrawSectionDialog — transport error", () => {
  it("shows inline transport message, no toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    const withdrawSection = vi.fn(async () => {
      throw new ConnectError("internal", Code.Internal);
    });

    renderDialog({ withdrawSection }, onOpenChange);

    await user.click(screen.getByRole("button", { name: /retirar/i }));

    await waitFor(() => expect(screen.getByRole("alert")).toBeInTheDocument());

    expect(screen.queryByText(/Internal/)).not.toBeInTheDocument();
    expect(toastError).not.toHaveBeenCalled();
    expect(onOpenChange).not.toHaveBeenCalledWith(false);
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
  });
});

describe("WithdrawSectionDialog — no auto-close on action click", () => {
  it("action click does not immediately close dialog", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    let resolve!: (
      v: ReturnType<typeof create<typeof WithdrawSectionResponseSchema>>,
    ) => void;
    const withdrawSection = vi.fn(
      () =>
        new Promise<
          ReturnType<typeof create<typeof WithdrawSectionResponseSchema>>
        >((r) => {
          resolve = r;
        }),
    );

    renderDialog({ withdrawSection }, onOpenChange);

    await user.click(screen.getByRole("button", { name: /retirar/i }));

    // Immediately after click, dialog is still open
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
    resolve(create(WithdrawSectionResponseSchema, {}));
  });
});

describe("WithdrawSectionDialog — display name resolution", () => {
  it("shows resolved student name in confirmation text when ProfileService returns a match", async () => {
    renderDialog({}, vi.fn(), {
      listDisplayNamesByIDs: async () => ({
        names: [
          {
            userId: "student-aaaaaaaa-0000-0000-0000-000000000001",
            givenNames: "Carlos",
            lastNamePaternal: "Ramírez",
          },
        ],
      }),
    });

    // The resolved display name should appear in the dialog description.
    expect(await screen.findByText(/Carlos Ramírez/)).toBeInTheDocument();
  });

  it("falls back to studentId[:8] in confirmation text when ProfileService returns no match", async () => {
    renderDialog({}, vi.fn(), {
      listDisplayNamesByIDs: async () => ({ names: [] }),
    });

    // The dialog renders with the UUID slice as the student identifier.
    expect(await screen.findByRole("alertdialog")).toBeInTheDocument();
    // studentId[:8] = "student-" (first 8 chars of "student-aaaaaaaa-0000-0000-0000-000000000001").
    expect(screen.getByText(/student-/)).toBeInTheDocument();
  });
});
