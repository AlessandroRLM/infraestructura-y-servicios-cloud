import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { SessionData } from "@/core/session";
import { renderWithProviders } from "@/core/test";

const unauthenticatedSession: SessionData = {
  user: null,
  roles: [],
  permissions: [],
  status: "unauthenticated",
};

const authenticatedSession: SessionData = {
  user: { id: "1", email: "user@test.com" },
  roles: ["student"],
  permissions: ["grades.read"],
  status: "authenticated",
};

describe("_authenticated route guard", () => {
  it("redirects to /login when session is unauthenticated", async () => {
    // The _authenticated index route is mounted at full path "/".
    // An unauthenticated session must trigger the beforeLoad redirect to /login.
    renderWithProviders(null, {
      route: "/",
      session: unauthenticatedSession,
    });
    const loginPage = await screen.findByTestId("login-page");
    expect(loginPage).toBeInTheDocument();
  });

  it("renders protected content when session is authenticated", async () => {
    renderWithProviders(null, {
      route: "/",
      session: authenticatedSession,
    });
    const dashboard = await screen.findByTestId("dashboard");
    expect(dashboard).toBeInTheDocument();
  });
});
