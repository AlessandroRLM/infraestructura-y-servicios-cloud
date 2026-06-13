import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "@/test";

describe("login route", () => {
  it("renders login placeholder", async () => {
    renderWithProviders({ route: "/login" });
    const loginPage = await screen.findByTestId("login-page");
    expect(loginPage).toBeInTheDocument();
  });
});
