/**
 * PayOwnEnrollmentDialog component tests.
 *
 * Covers:
 *  - Confirm calls markOwnEnrollmentPaid({id}) and shows success toast.
 *  - FailedPrecondition → inline precondition message (no raw code, no toast).
 *  - Transport error → inline transport message (no toast).
 *  - No auto-close on action click (e.preventDefault pattern).
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
import { renderComponent } from "@/test";
import { PayOwnEnrollmentDialog } from "../components/PayOwnEnrollmentDialog";

const { toastSuccess, toastError } = vi.hoisted(() => ({
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));
vi.mock("sonner", () => ({
  toast: { success: toastSuccess, error: toastError },
}));

type EnrollmentImpl = Partial<ServiceImpl<typeof EnrollmentService>>;

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

function renderDialog(impl: EnrollmentImpl, onOpenChange = vi.fn()) {
  renderComponent(
    <PayOwnEnrollmentDialog
      open
      onOpenChange={onOpenChange}
      enrollment={mockEnrollment}
    />,
    { transport: makeStubTransport([EnrollmentService, impl]) },
  );
}

describe("PayOwnEnrollmentDialog — confirm success", () => {
  it("calls markOwnEnrollmentPaid({id}) and shows success toast", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    const markOwnEnrollmentPaid = vi.fn(async () => mockEnrollment);

    renderDialog({ markOwnEnrollmentPaid }, onOpenChange);

    await user.click(screen.getByRole("button", { name: /pagar/i }));

    await waitFor(() =>
      expect(markOwnEnrollmentPaid).toHaveBeenCalledWith(
        expect.objectContaining({ id: "enroll-1" }),
        expect.anything(),
      ),
    );
    await waitFor(() => expect(toastSuccess).toHaveBeenCalled());
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
  });
});

describe("PayOwnEnrollmentDialog — FailedPrecondition", () => {
  it("shows inline precondition message, no toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    const markOwnEnrollmentPaid = vi.fn(async () => {
      throw new ConnectError("failed precondition", Code.FailedPrecondition);
    });

    renderDialog({ markOwnEnrollmentPaid }, onOpenChange);

    await user.click(screen.getByRole("button", { name: /pagar/i }));

    await waitFor(() => expect(screen.getByRole("alert")).toBeInTheDocument());

    expect(screen.queryByText(/FailedPrecondition/)).not.toBeInTheDocument();
    expect(screen.queryByText(/FAILED_PRECONDITION/)).not.toBeInTheDocument();
    expect(toastError).not.toHaveBeenCalled();
    expect(onOpenChange).not.toHaveBeenCalledWith(false);
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
  });
});

describe("PayOwnEnrollmentDialog — transport error", () => {
  it("shows inline transport message, no toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    const markOwnEnrollmentPaid = vi.fn(async () => {
      throw new ConnectError("internal", Code.Internal);
    });

    renderDialog({ markOwnEnrollmentPaid }, onOpenChange);

    await user.click(screen.getByRole("button", { name: /pagar/i }));

    await waitFor(() => expect(screen.getByRole("alert")).toBeInTheDocument());

    expect(screen.queryByText(/Internal/)).not.toBeInTheDocument();
    expect(toastError).not.toHaveBeenCalled();
    expect(onOpenChange).not.toHaveBeenCalledWith(false);
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
  });
});

describe("PayOwnEnrollmentDialog — no auto-close on action click", () => {
  it("action click does not immediately close dialog (e.preventDefault pattern)", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    let resolve!: (v: typeof mockEnrollment) => void;
    const markOwnEnrollmentPaid = vi.fn(
      () =>
        new Promise<typeof mockEnrollment>((r) => {
          resolve = r;
        }),
    );

    renderDialog({ markOwnEnrollmentPaid }, onOpenChange);

    await user.click(screen.getByRole("button", { name: /pagar/i }));

    // Immediately after click, dialog is still present
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
    // Resolve so cleanup doesn't hang
    resolve(mockEnrollment);
  });
});
