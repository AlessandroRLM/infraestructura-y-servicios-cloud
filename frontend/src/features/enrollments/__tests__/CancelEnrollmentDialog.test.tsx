/**
 * CancelEnrollmentDialog component tests.
 *
 * Covers:
 *  - Confirm calls cancelEnrollment({id}) and shows success toast.
 *  - FailedPrecondition → inline precondition message (no raw code, no toast).
 *  - Transport error → inline transport message (no toast).
 *  - No auto-close on action click (e.preventDefault pattern).
 *  - "Volver" cancel button label is present.
 */
import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import {
  CancelEnrollmentResponseSchema,
  EnrollmentSchema,
  EnrollmentService,
} from "@/gen/enrollment/v1/enrollment_pb";
import { renderComponent } from "@/test";
import { CancelEnrollmentDialog } from "../components/CancelEnrollmentDialog";

const { toastSuccess, toastError } = vi.hoisted(() => ({
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));
vi.mock("sonner", () => ({
  toast: { success: toastSuccess, error: toastError },
}));

type EnrollmentImpl = Partial<ServiceImpl<typeof EnrollmentService>>;

const mockEnrollment = create(EnrollmentSchema, {
  id: "enroll-2",
  studentId: "student-1",
  programId: "prog-1",
  year: 2025,
  status: "paid",
  createdAt: "",
  updatedAt: "",
  programName: "Ingeniería Industrial",
  studentName: "Carlos López",
});

function renderDialog(impl: EnrollmentImpl, onOpenChange = vi.fn()) {
  renderComponent(
    <CancelEnrollmentDialog
      open
      onOpenChange={onOpenChange}
      enrollment={mockEnrollment}
    />,
    { transport: makeStubTransport([EnrollmentService, impl]) },
  );
}

describe("CancelEnrollmentDialog — confirm success", () => {
  it("calls cancelEnrollment({id}) and shows success toast", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    const cancelEnrollment = vi.fn(async () =>
      create(CancelEnrollmentResponseSchema, {}),
    );

    renderDialog({ cancelEnrollment }, onOpenChange);

    await user.click(
      screen.getByRole("button", { name: /cancelar matrícula/i }),
    );

    await waitFor(() =>
      expect(cancelEnrollment).toHaveBeenCalledWith(
        expect.objectContaining({ id: "enroll-2" }),
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

describe("CancelEnrollmentDialog — FailedPrecondition", () => {
  it("shows inline precondition message, no toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    const cancelEnrollment = vi.fn(async () => {
      throw new ConnectError("failed precondition", Code.FailedPrecondition);
    });

    renderDialog({ cancelEnrollment }, onOpenChange);

    await user.click(
      screen.getByRole("button", { name: /cancelar matrícula/i }),
    );

    await waitFor(() =>
      expect(screen.getByRole("alert")).toBeInTheDocument(),
    );

    expect(screen.queryByText(/FailedPrecondition/)).not.toBeInTheDocument();
    expect(screen.queryByText(/FAILED_PRECONDITION/)).not.toBeInTheDocument();
    expect(toastError).not.toHaveBeenCalled();
    expect(onOpenChange).not.toHaveBeenCalledWith(false);
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
  });
});

describe("CancelEnrollmentDialog — transport error", () => {
  it("shows inline transport message, no toast, dialog stays open", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    const cancelEnrollment = vi.fn(async () => {
      throw new ConnectError("internal", Code.Internal);
    });

    renderDialog({ cancelEnrollment }, onOpenChange);

    await user.click(
      screen.getByRole("button", { name: /cancelar matrícula/i }),
    );

    await waitFor(() =>
      expect(screen.getByRole("alert")).toBeInTheDocument(),
    );

    expect(screen.queryByText(/Internal/)).not.toBeInTheDocument();
    expect(toastError).not.toHaveBeenCalled();
    expect(onOpenChange).not.toHaveBeenCalledWith(false);
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
  });
});

describe("CancelEnrollmentDialog — no auto-close on action click", () => {
  it("action click does not immediately close dialog", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    let resolve!: (v: ReturnType<typeof create<typeof CancelEnrollmentResponseSchema>>) => void;
    const cancelEnrollment = vi.fn(
      () =>
        new Promise<ReturnType<typeof create<typeof CancelEnrollmentResponseSchema>>>(
          (r) => { resolve = r; },
        ),
    );

    renderDialog({ cancelEnrollment }, onOpenChange);

    await user.click(
      screen.getByRole("button", { name: /cancelar matrícula/i }),
    );

    // Immediately after click, dialog is still open
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
    resolve(create(CancelEnrollmentResponseSchema, {}));
  });
});
