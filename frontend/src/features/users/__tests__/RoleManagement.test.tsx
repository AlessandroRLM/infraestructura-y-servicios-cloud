import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { TransportProvider } from "@connectrpc/connect-query";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import type { SessionState } from "@/features/auth";
import { SessionContext } from "@/features/auth";
import {
  AssignRoleResponseSchema,
  IamService,
  RevokeRoleResponseSchema,
  UserSummarySchema,
} from "@/gen/iam/v1/iam_pb";
import { RoleManagement } from "../components/RoleManagement";

const { toastSuccess, toastError } = vi.hoisted(() => ({
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));
vi.mock("sonner", () => ({
  toast: { success: toastSuccess, error: toastError },
}));

type IamImpl = Partial<ServiceImpl<typeof IamService>>;

const adminSession: SessionState = {
  status: "authenticated",
  userId: "u-admin",
  email: "admin@test.com",
  roles: ["admin"],
  permissions: ["users.manage"],
};

// Session where session.userId === the target user's id (self).
const selfSession: SessionState = {
  status: "authenticated",
  userId: "u-target",
  email: "self@test.com",
  roles: ["admin"],
  permissions: ["users.manage"],
};

const updatedUser = create(UserSummarySchema, {
  id: "u-target",
  email: "target@test.com",
  roles: ["student", "teacher"],
});

function renderComponent(
  iamHandlers: IamImpl,
  props: { userId: string; roles: string[] },
  session: SessionState = adminSession,
) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0, staleTime: 0 } },
  });
  const transport = makeStubTransport([IamService, iamHandlers]);

  return render(
    <TransportProvider transport={transport}>
      <QueryClientProvider client={queryClient}>
        <SessionContext value={session}>
          <RoleManagement {...props} />
        </SessionContext>
      </QueryClientProvider>
    </TransportProvider>,
  );
}

describe("RoleManagement", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("S-RM-01: roles rendered as ordered badges with revoke-X", () => {
    renderComponent({}, { userId: "u-target", roles: ["student", "teacher"] });

    // teacher comes before student (ROLE_PRIORITY order: admin→teacher→student).
    const badges = screen.getAllByText(/Profesor|Estudiante/);
    expect(badges[0]).toHaveTextContent("Profesor");
    expect(badges[1]).toHaveTextContent("Estudiante");

    // Both badges have a revoke-X (not self, not admin role).
    expect(
      screen.getByRole("button", { name: "Quitar rol Profesor" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Quitar rol Estudiante" }),
    ).toBeInTheDocument();

    // Add-role Select is present (user lacks admin).
    expect(screen.getByRole("combobox")).toBeInTheDocument();
  });

  it("S-RM-02: Add-role Select shows only roles not held", async () => {
    const user = userEvent.setup();
    renderComponent({}, { userId: "u-target", roles: ["teacher"] });

    await user.click(screen.getByRole("combobox"));

    await waitFor(() => {
      expect(
        screen.getByRole("option", { name: "Administrador" }),
      ).toBeInTheDocument();
      expect(
        screen.getByRole("option", { name: "Estudiante" }),
      ).toBeInTheDocument();
    });
    expect(
      screen.queryByRole("option", { name: "Profesor" }),
    ).not.toBeInTheDocument();
  });

  it("S-RM-03: Select hidden when user holds all three roles", () => {
    renderComponent(
      {},
      { userId: "u-target", roles: ["admin", "teacher", "student"] },
    );

    expect(screen.queryByRole("combobox")).not.toBeInTheDocument();
    // All three badges present.
    expect(screen.getByText("Administrador")).toBeInTheDocument();
    expect(screen.getByText("Profesor")).toBeInTheDocument();
    expect(screen.getByText("Estudiante")).toBeInTheDocument();
  });

  it("S-RM-04: assign role success — toast and no error shown", async () => {
    const user = userEvent.setup();
    const assignRole = vi.fn(async () =>
      create(AssignRoleResponseSchema, { user: updatedUser }),
    );

    renderComponent({ assignRole }, { userId: "u-target", roles: ["student"] });

    await user.click(screen.getByRole("combobox"));
    await waitFor(() =>
      expect(
        screen.getByRole("option", { name: "Profesor" }),
      ).toBeInTheDocument(),
    );
    await user.click(screen.getByRole("option", { name: "Profesor" }));

    await waitFor(() =>
      expect(toastSuccess).toHaveBeenCalledWith("Rol asignado"),
    );
    expect(assignRole).toHaveBeenCalledWith(
      expect.objectContaining({ userId: "u-target", role: "teacher" }),
      expect.anything(),
    );
    expect(toastError).not.toHaveBeenCalled();
  });

  it("S-RM-05: assign idempotent — backend success treated as success", async () => {
    const user = userEvent.setup();
    // Backend returns success even if role already held.
    const assignRole = vi.fn(async () =>
      create(AssignRoleResponseSchema, { user: updatedUser }),
    );

    renderComponent({ assignRole }, { userId: "u-target", roles: ["student"] });

    await user.click(screen.getByRole("combobox"));
    await waitFor(() =>
      expect(
        screen.getByRole("option", { name: "Profesor" }),
      ).toBeInTheDocument(),
    );
    await user.click(screen.getByRole("option", { name: "Profesor" }));

    await waitFor(() =>
      expect(toastSuccess).toHaveBeenCalledWith("Rol asignado"),
    );
    expect(toastError).not.toHaveBeenCalled();
  });

  it("S-RM-06: assign error — generic toast, no backend detail", async () => {
    const user = userEvent.setup();
    const assignRole = vi.fn(async () => {
      throw new ConnectError("internal", Code.Internal);
    });

    renderComponent({ assignRole }, { userId: "u-target", roles: ["student"] });

    await user.click(screen.getByRole("combobox"));
    await waitFor(() =>
      expect(
        screen.getByRole("option", { name: "Profesor" }),
      ).toBeInTheDocument(),
    );
    await user.click(screen.getByRole("option", { name: "Profesor" }));

    await waitFor(() =>
      expect(toastError).toHaveBeenCalledWith(
        "No se pudo completar la acción. Inténtalo de nuevo.",
      ),
    );
    expect(toastSuccess).not.toHaveBeenCalled();
    // Toast copy must not contain error codes or service names.
    const call = toastError.mock.calls[0][0] as string;
    expect(call).not.toMatch(/Internal|Code|ConnectError/i);
  });

  it("S-RM-07: click X opens AlertDialog without calling revokeRole", async () => {
    const user = userEvent.setup();
    const revokeRole = vi.fn();

    renderComponent(
      { revokeRole },
      { userId: "u-target", roles: ["admin", "student"] },
    );

    await user.click(
      screen.getByRole("button", { name: "Quitar rol Estudiante" }),
    );

    await waitFor(() =>
      expect(screen.getByRole("alertdialog")).toBeInTheDocument(),
    );
    expect(revokeRole).not.toHaveBeenCalled();
  });

  it("S-RM-08: revoke cancel — dialog closes, revokeRole not called", async () => {
    const user = userEvent.setup();
    const revokeRole = vi.fn();

    renderComponent({ revokeRole }, { userId: "u-target", roles: ["student"] });

    await user.click(
      screen.getByRole("button", { name: "Quitar rol Estudiante" }),
    );
    await waitFor(() =>
      expect(screen.getByRole("alertdialog")).toBeInTheDocument(),
    );

    await user.click(screen.getByRole("button", { name: "Cancelar" }));

    await waitFor(() =>
      expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument(),
    );
    expect(revokeRole).not.toHaveBeenCalled();
  });

  it("S-RM-09: revoke confirm success — toast, dialog closes, revokeRole called", async () => {
    const user = userEvent.setup();
    const revokeRole = vi.fn(async () => create(RevokeRoleResponseSchema, {}));

    renderComponent({ revokeRole }, { userId: "u-target", roles: ["student"] });

    await user.click(
      screen.getByRole("button", { name: "Quitar rol Estudiante" }),
    );
    await waitFor(() =>
      expect(screen.getByRole("alertdialog")).toBeInTheDocument(),
    );

    await user.click(screen.getByRole("button", { name: "Quitar" }));

    await waitFor(() =>
      expect(toastSuccess).toHaveBeenCalledWith("Rol revocado"),
    );
    await waitFor(() =>
      expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument(),
    );
    expect(revokeRole).toHaveBeenCalledWith(
      expect.objectContaining({ userId: "u-target", role: "student" }),
      expect.anything(),
    );
  });

  it("S-RM-10: revoke FailedPrecondition — dialog stays open, inline alert shown, no toast", async () => {
    const user = userEvent.setup();
    const revokeRole = vi.fn(async () => {
      throw new ConnectError("last admin", Code.FailedPrecondition);
    });

    renderComponent({ revokeRole }, { userId: "u-target", roles: ["student"] });

    await user.click(
      screen.getByRole("button", { name: "Quitar rol Estudiante" }),
    );
    await waitFor(() =>
      expect(screen.getByRole("alertdialog")).toBeInTheDocument(),
    );

    await user.click(screen.getByRole("button", { name: "Quitar" }));

    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent(
        "No se puede quitar el último administrador del sistema.",
      ),
    );
    // Dialog must remain open.
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
    expect(toastError).not.toHaveBeenCalled();
  });

  it("S-RM-11: revoke Internal error — toast shown, dialog closes", async () => {
    const user = userEvent.setup();
    const revokeRole = vi.fn(async () => {
      throw new ConnectError("internal error", Code.Internal);
    });

    renderComponent({ revokeRole }, { userId: "u-target", roles: ["student"] });

    await user.click(
      screen.getByRole("button", { name: "Quitar rol Estudiante" }),
    );
    await waitFor(() =>
      expect(screen.getByRole("alertdialog")).toBeInTheDocument(),
    );

    await user.click(screen.getByRole("button", { name: "Quitar" }));

    await waitFor(() =>
      expect(toastError).toHaveBeenCalledWith(
        "No se pudo completar la acción. Inténtalo de nuevo.",
      ),
    );
    await waitFor(() =>
      expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument(),
    );
  });

  it("S-RM-12: self-demotion — admin X hidden for own admin badge, other roles unaffected", () => {
    // selfSession.userId === userId === "u-target".
    renderComponent(
      {},
      { userId: "u-target", roles: ["admin", "student"] },
      selfSession,
    );

    // Admin badge is rendered.
    expect(screen.getByText("Administrador")).toBeInTheDocument();
    // Revoke-X for admin is NOT present.
    expect(
      screen.queryByRole("button", { name: "Quitar rol Administrador" }),
    ).not.toBeInTheDocument();
    // Student X is still present.
    expect(
      screen.getByRole("button", { name: "Quitar rol Estudiante" }),
    ).toBeInTheDocument();
  });

  it("S-RM-14: in-flight assign — all controls disabled (hard assertions)", async () => {
    // Mutation that never resolves so isPending stays true.
    let cancelFn!: () => void;
    const assignRole = vi.fn(
      () =>
        new Promise<never>((_resolve, reject) => {
          cancelFn = () => reject(new Error("cancelled"));
        }),
    );
    const user = userEvent.setup();

    // Use a teacher role so the revoke-X is guaranteed present and revocable.
    renderComponent({ assignRole }, { userId: "u-target", roles: ["teacher"] });

    // Trigger the mutation by selecting a role (Administrador).
    await user.click(screen.getByRole("combobox"));
    await waitFor(() =>
      expect(
        screen.getByRole("option", { name: "Administrador" }),
      ).toBeInTheDocument(),
    );
    await user.click(screen.getByRole("option", { name: "Administrador" }));

    // HARD assertion: combobox must be disabled while assign is in-flight.
    await waitFor(() => expect(screen.getByRole("combobox")).toBeDisabled());

    // HARD assertion: the revoke-X for Profesor must be IN the document AND disabled.
    expect(
      screen.getByRole("button", { name: "Quitar rol Profesor" }),
    ).toBeDisabled();

    // Flush the rejected promise so it doesn't leak into subsequent tests.
    await act(async () => {
      cancelFn();
      // Yield to microtask queue so the catch branch fires inside this act scope.
      await Promise.resolve();
    });
  });

  it("S-RM-13: self-demotion backstop — FailedPrecondition on non-admin role shows inline error", async () => {
    const user = userEvent.setup();
    const revokeRole = vi.fn(async () => {
      throw new ConnectError("last admin", Code.FailedPrecondition);
    });

    // selfSession.userId === "u-target" === userId.
    // Admin-X is pre-empted (canRevoke returns false for admin on self).
    // Teacher X IS shown — revoke triggers the backstop path on FailedPrecondition.
    renderComponent(
      { revokeRole },
      { userId: "u-target", roles: ["admin", "teacher"] },
      selfSession,
    );

    // Admin-X is hidden for self (pre-empt guard), teacher-X must be present.
    expect(
      screen.queryByRole("button", { name: "Quitar rol Administrador" }),
    ).not.toBeInTheDocument();
    const teacherRevoke = screen.getByRole("button", {
      name: "Quitar rol Profesor",
    });
    expect(teacherRevoke).toBeInTheDocument();

    // Open the confirmation dialog.
    await user.click(teacherRevoke);
    await waitFor(() =>
      expect(screen.getByRole("alertdialog")).toBeInTheDocument(),
    );

    // Confirm the revoke — backend throws FailedPrecondition.
    await user.click(screen.getByRole("button", { name: "Quitar" }));

    // Backstop: inline alert inside the open dialog must appear.
    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent(
        "No se puede quitar el último administrador del sistema.",
      ),
    );
    // Dialog must remain open.
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
    // No toast for FailedPrecondition.
    expect(toastError).not.toHaveBeenCalled();
  });

  it("S-RM-16: DISABLED user — role controls are present and usable", async () => {
    const user = userEvent.setup();
    const assignRole = vi.fn(async () =>
      create(AssignRoleResponseSchema, { user: updatedUser }),
    );

    // RoleManagement has no status prop — it cannot block on user status.
    // Render with a user that is conceptually DISABLED (status not passed to component).
    renderComponent({ assignRole }, { userId: "u-target", roles: ["teacher"] });

    // Revoke-X is present for the teacher role.
    expect(
      screen.getByRole("button", { name: "Quitar rol Profesor" }),
    ).toBeInTheDocument();

    // Add-role Select (combobox) is rendered and usable.
    const combobox = screen.getByRole("combobox");
    expect(combobox).toBeInTheDocument();
    expect(combobox).not.toBeDisabled();

    // Assigning a role succeeds — no client-side block on disabled status.
    await user.click(combobox);
    await waitFor(() =>
      expect(
        screen.getByRole("option", { name: "Estudiante" }),
      ).toBeInTheDocument(),
    );
    await user.click(screen.getByRole("option", { name: "Estudiante" }));

    await waitFor(() =>
      expect(toastSuccess).toHaveBeenCalledWith("Rol asignado"),
    );
    expect(assignRole).toHaveBeenCalledWith(
      expect.objectContaining({ userId: "u-target", role: "student" }),
      expect.anything(),
    );
  });

  it("S-RM-15: empty roles — shows empty-state text and full role list in Select", async () => {
    const user = userEvent.setup();
    renderComponent({}, { userId: "u-target", roles: [] });

    expect(screen.getByText("Sin roles asignados")).toBeInTheDocument();
    // No badge chips.
    expect(screen.queryByText("Administrador")).not.toBeInTheDocument();
    expect(screen.queryByText("Profesor")).not.toBeInTheDocument();
    expect(screen.queryByText("Estudiante")).not.toBeInTheDocument();

    // All three roles in the Select.
    await user.click(screen.getByRole("combobox"));
    await waitFor(() => {
      expect(
        screen.getByRole("option", { name: "Administrador" }),
      ).toBeInTheDocument();
      expect(
        screen.getByRole("option", { name: "Profesor" }),
      ).toBeInTheDocument();
      expect(
        screen.getByRole("option", { name: "Estudiante" }),
      ).toBeInTheDocument();
    });
  });

  it("S-RM-17: badge order follows ROLE_PRIORITY regardless of server order", () => {
    renderComponent(
      {},
      { userId: "u-target", roles: ["student", "admin", "teacher"] },
    );

    const badges = screen.getAllByText(/Administrador|Profesor|Estudiante/);
    expect(badges[0]).toHaveTextContent("Administrador");
    expect(badges[1]).toHaveTextContent("Profesor");
    expect(badges[2]).toHaveTextContent("Estudiante");
  });
});
