import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { makeStubTransport } from "@/core/test";
import { CatalogService } from "@/gen/catalog/v1/catalog_pb";
import { renderWithProviders } from "@/test";
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
