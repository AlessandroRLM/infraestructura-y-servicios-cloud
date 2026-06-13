import { create } from "@bufbuild/protobuf";
import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import type { SessionSource } from "@/features/auth";
import { AuthService, LoginResponseSchema } from "@/gen/auth/v1/auth_pb";
import { renderWithProviders } from "@/test";

const { toastError } = vi.hoisted(() => ({ toastError: vi.fn() }));
vi.mock("sonner", () => ({ toast: { error: toastError } }));

type LoginHandler = ServiceImpl<typeof AuthService>["login"];

async function renderLogin(login: LoginHandler, sessionSource?: SessionSource) {
  renderWithProviders({
    route: "/login",
    transport: makeStubTransport([AuthService, { login }]),
    ...(sessionSource ? { sessionSource } : {}),
  });
  // The router resolves the route asynchronously; wait for the form to mount.
  await screen.findByRole("button", { name: "Sign in" });
}

describe("LoginForm", () => {
  it("shows a validation error for an invalid email and does not call the RPC", async () => {
    const user = userEvent.setup();
    const login = vi.fn(async () => create(LoginResponseSchema, {}));
    await renderLogin(login);

    await user.type(screen.getByLabelText("Email"), "not-an-email");
    await user.type(screen.getByLabelText("Password"), "secret");
    await user.click(screen.getByRole("button", { name: "Sign in" }));

    expect(await screen.findByText("Enter a valid email")).toBeInTheDocument();
    expect(login).not.toHaveBeenCalled();
  });

  it("calls the login RPC with the entered credentials", async () => {
    const user = userEvent.setup();
    const login = vi.fn(async (req: { email: string; password: string }) => {
      void req;
      return create(LoginResponseSchema, {});
    });
    await renderLogin(login);

    await user.type(screen.getByLabelText("Email"), "user@test.com");
    await user.type(screen.getByLabelText("Password"), "secret");
    await user.click(screen.getByRole("button", { name: "Sign in" }));

    await waitFor(() => expect(login).toHaveBeenCalledTimes(1));
    expect(login.mock.calls[0][0]).toMatchObject({
      email: "user@test.com",
      password: "secret",
    });
  });

  it("navigates to the dashboard after a successful login", async () => {
    const user = userEvent.setup();
    const login = vi.fn(async () => create(LoginResponseSchema, {}));

    // After login the hook invalidates SESSION_QUERY_KEY; the guard refetches
    // via the injected source, gets an authenticated session, and renders the dashboard.
    const sessionSource: SessionSource = {
      getSession: async () => ({
        userId: "1",
        email: "user@test.com",
        roles: ["student"],
        permissions: [],
      }),
    };

    await renderLogin(login, sessionSource);

    await user.type(screen.getByLabelText("Email"), "user@test.com");
    await user.type(screen.getByLabelText("Password"), "secret");
    await user.click(screen.getByRole("button", { name: "Sign in" }));

    expect(await screen.findByTestId("dashboard")).toBeInTheDocument();
  });

  it("shows inline error when credentials are wrong (CodeUnauthenticated)", async () => {
    const user = userEvent.setup();
    const login = vi.fn(async () => {
      throw new ConnectError("bad creds", Code.Unauthenticated);
    });
    await renderLogin(login);

    await user.type(screen.getByLabelText("Email"), "user@test.com");
    await user.type(screen.getByLabelText("Password"), "wrong");
    await user.click(screen.getByRole("button", { name: "Sign in" }));

    expect(
      await screen.findByText("Email or password is incorrect"),
    ).toBeInTheDocument();
    // Confirm the alert role carries the correct message.
    expect(screen.getByRole("alert")).toHaveTextContent(
      "Email or password is incorrect",
    );
    expect(toastError).not.toHaveBeenCalled();
  });

  it("shows a toast when a non-auth error occurs (e.g. Code.Internal)", async () => {
    const user = userEvent.setup();
    const login = vi.fn(async () => {
      throw new ConnectError("server error", Code.Internal);
    });
    await renderLogin(login);

    await user.type(screen.getByLabelText("Email"), "user@test.com");
    await user.type(screen.getByLabelText("Password"), "secret");
    await user.click(screen.getByRole("button", { name: "Sign in" }));

    await waitFor(() =>
      expect(toastError).toHaveBeenCalledWith(
        "Couldn't connect. Please try again.",
      ),
    );
    // Toast path must NOT also set the inline error.
    expect(
      screen.queryByText("Email or password is incorrect"),
    ).not.toBeInTheDocument();
  });

  it("redirects to the redirect search param destination after successful login", async () => {
    const user = userEvent.setup();
    const login = vi.fn(async () => create(LoginResponseSchema, {}));

    // After login the guard refetches via the injected source; authenticated
    // session causes it to navigate to the redirect target instead of "/".
    const sessionSource: SessionSource = {
      getSession: async () => ({
        userId: "1",
        email: "user@test.com",
        roles: ["student"],
        permissions: [],
      }),
    };

    const { router } = renderWithProviders({
      route: "/login?redirect=%2Fsettings",
      transport: makeStubTransport([AuthService, { login }]),
      sessionSource,
    });

    await user.type(await screen.findByLabelText("Email"), "user@test.com");
    await user.type(screen.getByLabelText("Password"), "secret");
    await user.click(screen.getByRole("button", { name: "Sign in" }));

    // Verify the router honored the injected redirect value, not the default "/".
    await waitFor(() =>
      expect(router.state.location.pathname).toBe("/settings"),
    );
  });
});
