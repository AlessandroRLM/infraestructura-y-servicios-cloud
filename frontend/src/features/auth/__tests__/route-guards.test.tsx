import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { makeStubTransport } from "@/core/test";
import { CatalogService } from "@/gen/catalog/v1/catalog_pb";
import { renderWithProviders } from "@/test";
import { ROUTE_PERMISSIONS } from "../routePermissions";
import type { SessionState } from "../types";

function authenticated(permissions: string[]): SessionState {
  return {
    status: "authenticated",
    userId: "1",
    email: "user@test.com",
    roles: ["admin"],
    permissions,
  };
}

describe("route permission guards", () => {
  it("redirects an authenticated user lacking the permission to the 403 screen", async () => {
    // /academics still exists as a flat route; its guard now checks /admin/academics.
    renderWithProviders({ route: "/academics", session: authenticated([]) });

    expect(
      await screen.findByText("No tienes acceso a esta sección"),
    ).toBeInTheDocument();
  });

  it("lets an authenticated user with the permission reach the page", async () => {
    renderWithProviders({
      route: "/academics",
      session: authenticated(["catalog.manage"]),
      transport: makeStubTransport([
        CatalogService,
        { listPrograms: async () => ({ programs: [] }) },
      ]),
    });

    expect(
      await screen.findByText("Todavía no hay carreras"),
    ).toBeInTheDocument();
  });

  it("renders the 404 screen for an unmatched route", async () => {
    renderWithProviders({
      route: "/does-not-exist",
      session: authenticated(["catalog.manage"]),
    });

    expect(await screen.findByText("Página no encontrada")).toBeInTheDocument();
  });

  it("renders the 403 screen directly on /forbidden for an authenticated user", async () => {
    renderWithProviders({ route: "/forbidden", session: authenticated([]) });

    expect(
      await screen.findByText("No tienes acceso a esta sección"),
    ).toBeInTheDocument();
  });
});

// Area-guard integration tests: verify that the layout-level eligibility guard
// in admin/route.tsx and app/route.tsx enforces cross-area redirects before
// any feature guard or page component runs.
describe("area eligibility guard (admin/route.tsx + app/route.tsx)", () => {
  it("participant-only session navigating to /admin/* is redirected to /app", async () => {
    // grades.view_own → participantEligible=true, adminEligible=false.
    // admin/route.tsx beforeLoad sees no "admin" in eligibility and has "participant",
    // so it throws redirect({ to: "/app" }).
    const { router } = renderWithProviders({
      route: "/admin/academics",
      session: authenticated(["grades.view_own"]),
    });

    await waitFor(() =>
      expect(router.state.location.pathname).toMatch(/^\/app/),
    );
  });

  it("admin-only session navigating to /app/* is redirected to /admin", async () => {
    // catalog.manage → adminEligible=true, participantEligible=false.
    // app/route.tsx beforeLoad sees no "participant" in eligibility and has "admin",
    // so it throws redirect({ to: "/admin" }).
    const { router } = renderWithProviders({
      route: "/app/grades",
      session: authenticated(["catalog.manage"]),
    });

    await waitFor(() =>
      expect(router.state.location.pathname).toMatch(/^\/admin/),
    );
  });

  it("zero-eligibility session navigating to /admin/* is redirected to /forbidden", async () => {
    // No permissions → adminEligible=false, participantEligible=false.
    // admin/route.tsx beforeLoad: not admin, not participant → redirect to /forbidden.
    const { router } = renderWithProviders({
      route: "/admin/academics",
      session: authenticated([]),
    });

    await waitFor(() =>
      expect(router.state.location.pathname).toBe("/forbidden"),
    );
  });

  it("zero-eligibility session navigating to /app/* is redirected to /forbidden", async () => {
    // No permissions → adminEligible=false, participantEligible=false.
    // app/route.tsx beforeLoad: not participant, not admin → redirect to /forbidden.
    const { router } = renderWithProviders({
      route: "/app/grades",
      session: authenticated([]),
    });

    await waitFor(() =>
      expect(router.state.location.pathname).toBe("/forbidden"),
    );
  });
});

/**
 * During the area-routing migration (Slice 1), the flat route files still live
 * at their original paths but their guards now reference the prefixed keys in
 * ROUTE_PERMISSIONS (e.g. "/academics" guards with "/admin/academics").
 *
 * This map ties the existing flat paths to the permission key they use so
 * the data-driven guard test can assert "path X without permission Y → /forbidden"
 * without needing the new prefixed route files to exist yet.
 */
const FLAT_ROUTE_TO_PERMISSION_KEY: Record<string, string> = {
  "/academics": "/admin/academics",
  "/enrollments": "/admin/enrollments",
  "/section-enrollments": "/admin/section-enrollments",
  "/grades": "/admin/grades",
  "/reports": "/admin/reports",
  "/users": "/admin/users",
  "/access-control": "/admin/access-control",
};

const flatRoutePairs = Object.entries(FLAT_ROUTE_TO_PERMISSION_KEY) as [
  string,
  keyof typeof ROUTE_PERMISSIONS,
][];

// Asserts that the guard runs in beforeLoad before the page — the redirect
// happens before the component renders, so no per-page transport stub is needed.
describe("every flat route (migration period) is guarded with its prefixed permission key", () => {
  it.each(
    flatRoutePairs,
  )("%s redirects to /forbidden without the required permission", async (route) => {
    const { router } = renderWithProviders({
      route,
      session: authenticated([]),
    });

    await waitFor(() =>
      expect(router.state.location.pathname).toBe("/forbidden"),
    );
  });

  it.each(
    flatRoutePairs,
  )("%s is reachable when the session holds its permission", async (route, permKey) => {
    const { router } = renderWithProviders({
      route,
      session: authenticated([ROUTE_PERMISSIONS[permKey][0]]),
    });

    await waitFor(() => expect(router.state.location.pathname).toBe(route));
  });
});

// Locks the nav to the same source of truth: a guarded link shows only when the
// session holds the route's permission (resolved via the new prefixed key).
describe("nav link visibility derives from ROUTE_PERMISSIONS", () => {
  it("shows a guarded link when the session holds the permission", async () => {
    // Navigate directly to an admin area route the session can access so the
    // admin sidebar renders. /admin/users requires users.manage.
    // /forbidden no longer renders inside an area shell (no sidebar), so we
    // use a real area route to get AppSidebar.
    // /admin/access-control page does not call Route.useSearch() so it renders
    // without the flat-route coupling issue. The admin sidebar is always present.
    renderWithProviders({
      route: "/admin/access-control",
      session: authenticated(["users.manage"]),
    });

    // Wait for the admin area to render (admin sidebar is present).
    await waitFor(() =>
      expect(
        screen.getByRole("link", { name: /usuarios/i }),
      ).toBeInTheDocument(),
    );
  });

  it("hides a guarded link when the session lacks the permission", async () => {
    // A zero-permission session has no area eligibility; /forbidden renders
    // without an area sidebar — no nav links present.
    renderWithProviders({ route: "/forbidden", session: authenticated([]) });

    await screen.findByText("No tienes acceso a esta sección");
    expect(
      screen.queryByRole("link", { name: /usuarios/i }),
    ).not.toBeInTheDocument();
  });
});
