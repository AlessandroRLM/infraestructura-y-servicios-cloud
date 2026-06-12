import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "@/core/test";

describe("login route", () => {
  it("renders login placeholder", async () => {
    renderWithProviders(null, { route: "/login" });
    const loginPage = await screen.findByTestId("login-page");
    expect(loginPage).toBeInTheDocument();
  });
});
