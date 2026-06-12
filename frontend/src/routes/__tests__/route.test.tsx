import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { SessionState } from "@/features/auth";
import { renderWithProviders } from "@/test";

const unauthenticatedSession: SessionState = { status: "unauthenticated" };

const authenticatedSession: SessionState = {
  status: "authenticated",
  userId: "1",
  email: "user@test.com",
  roles: ["student"],
  permissions: ["grades.read"],
};

describe("_authenticated route guard", () => {
  it("redirects to /login when session is unauthenticated", async () => {
    renderWithProviders({
      route: "/",
      session: unauthenticatedSession,
    });

    expect(await screen.findByTestId("login-page")).toBeInTheDocument();
  });

  it("renders protected content when session is authenticated", async () => {
    renderWithProviders({
      route: "/",
      session: authenticatedSession,
    });

    expect(await screen.findByTestId("dashboard")).toBeInTheDocument();
  });

  it("redirects to /login when session is loading (cache unseeded, stub resolves null)", async () => {
    renderWithProviders({
      route: "/",
      session: { status: "loading" },
    });

    expect(await screen.findByTestId("login-page")).toBeInTheDocument();
  });
});
