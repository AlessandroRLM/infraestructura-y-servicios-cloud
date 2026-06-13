import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import { SESSION_QUERY_KEY } from "@/features/auth";
import { AuthService, LogoutResponseSchema } from "@/gen/auth/v1/auth_pb";
import { renderWithProviders } from "@/test";

const { toastError } = vi.hoisted(() => ({ toastError: vi.fn() }));
vi.mock("sonner", () => ({ toast: { error: toastError } }));

const authenticatedSession = {
  status: "authenticated" as const,
  userId: "1",
  email: "user@test.com",
  roles: ["student"] as string[],
  permissions: [] as string[],
};

describe("LogoutButton", () => {
  it("calls the logout RPC, resets the session cache, and redirects to /login", async () => {
    const user = userEvent.setup();
    const logout = vi.fn(async () => create(LogoutResponseSchema, {}));

    const { queryClient } = renderWithProviders({
      route: "/",
      session: authenticatedSession,
      transport: makeStubTransport([AuthService, { logout }]),
    });

    // Spy on setQueryData to capture the cleared value before gcTime:0 GCs it.
    const setQueryDataSpy = vi.spyOn(queryClient, "setQueryData");

    await user.click(
      await screen.findByRole("button", { name: "Cerrar sesión" }),
    );

    await waitFor(() => expect(logout).toHaveBeenCalledTimes(1));

    // Verify the cache was explicitly set to null (gcTime:0 GCs it after unmount,
    // so we check the spy rather than the post-navigation cache state).
    expect(setQueryDataSpy).toHaveBeenCalledWith(SESSION_QUERY_KEY, null);

    expect(await screen.findByTestId("login-page")).toBeInTheDocument();
  });

  it("clears cache and redirects to /login when RPC throws Unauthenticated (expired cookie)", async () => {
    const user = userEvent.setup();
    const logout = vi.fn(async () => {
      throw new ConnectError("expired", Code.Unauthenticated);
    });

    const { queryClient } = renderWithProviders({
      route: "/",
      session: authenticatedSession,
      transport: makeStubTransport([AuthService, { logout }]),
    });

    const setQueryDataSpy = vi.spyOn(queryClient, "setQueryData");

    await user.click(
      await screen.findByRole("button", { name: "Cerrar sesión" }),
    );

    // Hook's onError clears the stale cache entry for the expired session.
    await waitFor(() =>
      expect(setQueryDataSpy).toHaveBeenCalledWith(SESSION_QUERY_KEY, null),
    );

    expect(await screen.findByTestId("login-page")).toBeInTheDocument();
    expect(toastError).not.toHaveBeenCalled();
  });

  it("shows toast and stays on dashboard when RPC throws a non-Unauthenticated error", async () => {
    const user = userEvent.setup();
    const logout = vi.fn(async () => {
      throw new ConnectError("boom", Code.Internal);
    });

    const { queryClient } = renderWithProviders({
      route: "/",
      session: authenticatedSession,
      transport: makeStubTransport([AuthService, { logout }]),
    });

    const setQueryDataSpy = vi.spyOn(queryClient, "setQueryData");

    await user.click(
      await screen.findByRole("button", { name: "Cerrar sesión" }),
    );

    await waitFor(() =>
      expect(toastError).toHaveBeenCalledWith(
        "No se pudo cerrar sesión. Inténtalo de nuevo.",
      ),
    );

    // Component stays on the dashboard — the Sign out button is still present.
    expect(
      screen.getByRole("button", { name: "Cerrar sesión" }),
    ).toBeInTheDocument();
    expect(screen.queryByTestId("login-page")).not.toBeInTheDocument();

    // Cache must NOT be cleared for non-auth errors.
    expect(setQueryDataSpy).not.toHaveBeenCalledWith(SESSION_QUERY_KEY, null);
  });
});
