import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { makeStubTransport } from "@/core/test";
import { CatalogService } from "@/gen/catalog/v1/catalog_pb";
import { renderWithProviders } from "@/test";
import { ROUTE_PERMISSIONS } from "../routePermissions";
import type { SessionState } from "../types";

const guardedRoutes = Object.keys(
  ROUTE_PERMISSIONS,
) as (keyof typeof ROUTE_PERMISSIONS)[];

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

// Data-driven over the map so a route added to ROUTE_PERMISSIONS without a
// matching beforeLoad (fail-open), or wired to the wrong permission key, fails
// here instead of shipping. Asserts the post-navigation pathname, so it needs
// no per-page transport stub — the guard runs in beforeLoad, before the page.
describe("every route in ROUTE_PERMISSIONS is actually guarded", () => {
  it.each(
    guardedRoutes,
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
    guardedRoutes,
  )("%s is reachable when the session holds its permission", async (route) => {
    const { router } = renderWithProviders({
      route,
      session: authenticated([ROUTE_PERMISSIONS[route][0]]),
    });

    await waitFor(() => expect(router.state.location.pathname).toBe(route));
  });
});

// Locks the nav to the same source of truth: a guarded link shows only when the
// session holds the route's permission.
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
