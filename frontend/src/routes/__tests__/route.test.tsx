import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { SessionState } from "@/features/auth";
import { renderWithProviders } from "@/test";

const unauthenticatedSession: SessionState = { status: "unauthenticated" };

// grades.read → admin-eligible only → single area → redirects to /admin
const adminSession: SessionState = {
  status: "authenticated",
  userId: "1",
  email: "user@test.com",
  roles: ["admin"],
  permissions: ["grades.read"],
};

// grades.view_own → participant-eligible only → single area → redirects to /app
const studentSession: SessionState = {
  status: "authenticated",
  userId: "2",
  email: "student@test.com",
  roles: ["student"],
  permissions: ["grades.view_own"],
};

// grades.write → dual-eligible, no stored preference → /choose-area
const teacherSession: SessionState = {
  status: "authenticated",
  userId: "3",
  email: "teacher@test.com",
  roles: ["teacher"],
  permissions: ["grades.write"],
};

describe("_authenticated route guard", () => {
  it("redirects to /login when session is unauthenticated", async () => {
    renderWithProviders({
      route: "/",
      session: unauthenticatedSession,
    });

    expect(await screen.findByTestId("login-page")).toBeInTheDocument();
  });

  it("redirects to /login when session is loading (cache unseeded, stub resolves null)", async () => {
    renderWithProviders({
      route: "/",
      session: { status: "loading" },
    });

    expect(await screen.findByTestId("login-page")).toBeInTheDocument();
  });
});

describe("_authenticated index redirect (T-08)", () => {
  it("admin-only session redirects to /admin", async () => {
    const { router } = renderWithProviders({
      route: "/",
      session: adminSession,
    });

    await waitFor(() => expect(router.state.location.pathname).toBe("/admin"));
  });

  it("participant-only session redirects to /app", async () => {
    const { router } = renderWithProviders({
      route: "/",
      session: studentSession,
    });

    await waitFor(() => expect(router.state.location.pathname).toBe("/app"));
  });

  it("dual-eligible session with no stored preference redirects to /choose-area", async () => {
    const { router } = renderWithProviders({
      route: "/",
      session: teacherSession,
    });

    await waitFor(() =>
      expect(router.state.location.pathname).toBe("/choose-area"),
    );
  });

  it("zero-eligibility session redirects to /forbidden", async () => {
    const { router } = renderWithProviders({
      route: "/",
      session: {
        status: "authenticated",
        userId: "4",
        email: "nobody@test.com",
        roles: [],
        permissions: [],
      },
    });

    await waitFor(() =>
      expect(router.state.location.pathname).toBe("/forbidden"),
    );
  });
});
