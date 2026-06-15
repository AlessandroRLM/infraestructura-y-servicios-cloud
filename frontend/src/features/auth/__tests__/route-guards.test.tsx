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

/**
 * During the area-routing migration (Slice 1), the flat route files still live
 * at their original paths but their guards now reference the prefixed keys in
 * ROUTE_PERMISSIONS (e.g. "/academics" guards with "/admin/academics").
 *
 * This map ties the existing flat paths to the permission key they use so
 * the data-driven guard test can assert "path X without permission Y → /forbidden"
 * without needing the new prefixed route files to exist yet.
 *
 * Slice 2 replaces this map with a test over the new /admin/* and /app/* paths.
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
    renderWithProviders({
      route: "/",
      session: authenticated(["users.manage"]),
    });

    expect(
      await screen.findByRole("link", { name: /usuarios/i }),
    ).toBeInTheDocument();
  });

  it("hides a guarded link when the session lacks the permission", async () => {
    renderWithProviders({ route: "/", session: authenticated([]) });

    await screen.findByTestId("dashboard");
    expect(
      screen.queryByRole("link", { name: /usuarios/i }),
    ).not.toBeInTheDocument();
  });
});
