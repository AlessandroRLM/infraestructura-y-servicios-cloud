/**
 * Integration tests: /profile renders within the area shell (sidebar present)
 * and provides a "Volver" back-navigation link pointing to the correct area home.
 *
 * Uses renderWithProviders at the /profile route so the full shell + page
 * component tree mounts, matching production behaviour.
 */

import type { ServiceImpl } from "@connectrpc/connect";
import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import type { AuthenticatedSession } from "@/features/auth";
import { ProfileService } from "@/gen/profiles/v1/profiles_pb";
import { renderWithProviders } from "@/test";

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

type ProfileImpl = Partial<ServiceImpl<typeof ProfileService>>;

// Never-resolving handler — keeps the profile query in loading state so we
// only need the shell to appear, not full page content.
const pendingHandler: ProfileImpl = {
  // biome-ignore lint/suspicious/noEmptyBlockStatements: intentional never-resolving promise for loading state
  getOwnProfile: () => new Promise(() => {}),
};

function makeSession(
  permissions: string[],
): AuthenticatedSession & { status: "authenticated" } {
  return {
    status: "authenticated",
    userId: "user-test",
    email: "test@example.com",
    roles: ["student"],
    permissions,
  };
}

function makeSessionSource(session: ReturnType<typeof makeSession>) {
  return {
    getSession: async (): Promise<AuthenticatedSession> => ({
      userId: session.userId,
      email: session.email,
      roles: session.roles,
      permissions: session.permissions,
    }),
  };
}

describe("Profile shell", () => {
  it("participant-only session: renders sidebar with participant nav links and back link to /app", async () => {
    const session = makeSession([
      "profile.view_own",
      "profile.edit_own",
      "enrollment.view_own",
      "grades.view_own",
    ]);

    renderWithProviders({
      route: "/profile",
      transport: makeStubTransport([ProfileService, pendingHandler]),
      session,
      sessionSource: makeSessionSource(session),
    });

    // Sidebar nav entries for participant area must be present
    expect(await screen.findByText("Mis notas")).toBeInTheDocument();
    expect(screen.getByText("Mis matrículas")).toBeInTheDocument();

    // Back link must point to /app (participant home)
    const backLink = screen.getByRole("link", { name: /volver/i });
    expect(backLink).toHaveAttribute("href", "/app");
  });

  it("admin-only session: renders sidebar with admin nav links and back link to /admin", async () => {
    const session = makeSession([
      "catalog.manage",
      "users.manage",
      "enrollment.manage",
      "grades.read",
      "reports.read",
    ]);

    renderWithProviders({
      route: "/profile",
      transport: makeStubTransport([ProfileService, pendingHandler]),
      session,
      sessionSource: makeSessionSource(session),
    });

    // Sidebar nav link for "Académico" (/admin/academics) must be present.
    // (The sidebar brand text is also "Académico", so query by link role to be specific.)
    await screen.findByRole("link", { name: /académico/i });
    expect(
      screen.getByRole("link", { name: /matrículas/i }),
    ).toBeInTheDocument();

    // Back link must point to /admin
    const backLink = screen.getByRole("link", { name: /volver/i });
    expect(backLink).toHaveAttribute("href", "/admin");
  });

  it("dual-eligible session: defaults to admin area sidebar and back link to /admin", async () => {
    // grades.write makes the user eligible for both areas
    const session = makeSession([
      "catalog.manage",
      "grades.write",
      "grades.view_own",
      "enrollment.view_own",
    ]);

    renderWithProviders({
      route: "/profile",
      transport: makeStubTransport([ProfileService, pendingHandler]),
      session,
      sessionSource: makeSessionSource(session),
    });

    // Admin nav link appears (admin is listed first in eligibleAreas, no stored preference)
    await screen.findByRole("link", { name: /académico/i });

    // Back link resolves to /admin
    const backLink = screen.getByRole("link", { name: /volver/i });
    expect(backLink).toHaveAttribute("href", "/admin");
  });

  it("all states render the Volver link (loaded state)", async () => {
    const session = makeSession([
      "profile.view_own",
      "profile.edit_own",
      "enrollment.view_own",
      "grades.view_own",
    ]);

    const handlers: ProfileImpl = {
      getOwnProfile: async () => ({
        userId: "user-test",
        givenNames: "Ana",
        lastNamePaternal: "López",
        nationalIdType: "RUT",
        nationalId: "11111111-1",
      }),
    };

    renderWithProviders({
      route: "/profile",
      transport: makeStubTransport([ProfileService, handlers]),
      session,
      sessionSource: makeSessionSource(session),
    });

    // Wait for loaded state
    await screen.findByText("Ana");

    // Back link present in loaded state too
    const backLink = screen.getByRole("link", { name: /volver/i });
    expect(backLink).toHaveAttribute("href", "/app");
  });
});
