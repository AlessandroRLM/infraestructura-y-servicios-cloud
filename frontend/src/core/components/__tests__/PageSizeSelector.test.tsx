import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { PageSizeSelector } from "../PageSizeSelector";

describe("PageSizeSelector", () => {
  it("S-PS-01: renders the current value in the trigger", () => {
    render(<PageSizeSelector value={20} onChange={vi.fn()} />);
    expect(screen.getByRole("combobox")).toBeInTheDocument();
    expect(screen.getByText("20 por página")).toBeInTheDocument();
  });

  it("S-PS-02: opens the dropdown and lists all three options", async () => {
    const user = userEvent.setup();
    render(<PageSizeSelector value={20} onChange={vi.fn()} />);

    await user.click(screen.getByRole("combobox"));

    expect(await screen.findByText("50 por página")).toBeInTheDocument();
    expect(screen.getByText("100 por página")).toBeInTheDocument();
  });

  it("S-PS-03: selecting an option calls onChange with the correct number", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<PageSizeSelector value={20} onChange={onChange} />);

    await user.click(screen.getByRole("combobox"));
    await user.click(await screen.findByText("50 por página"));

    expect(onChange).toHaveBeenCalledWith(50);
  });

  it("S-PS-04: custom options renders only the provided values", async () => {
    const user = userEvent.setup();
    render(
      <PageSizeSelector value={10} onChange={vi.fn()} options={[10, 25]} />,
    );

    await user.click(screen.getByRole("combobox"));

    // Both the trigger value and the option in the dropdown show "10 por página".
    const tenPerPage = await screen.findAllByText("10 por página");
    expect(tenPerPage.length).toBeGreaterThanOrEqual(1);
    expect(await screen.findByText("25 por página")).toBeInTheDocument();
    expect(screen.queryByText("50 por página")).not.toBeInTheDocument();
  });
});
