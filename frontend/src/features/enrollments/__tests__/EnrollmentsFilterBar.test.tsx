/**
 * EnrollmentsFilterBar component tests.
 *
 * Covers:
 *  - Renders SearchInput with correct placeholder.
 *  - SearchInput onChange fires with debounced value.
 *  - Year input: valid year calls onYearChange; out-of-range keeps it undefined.
 *  - Status Select: __all__ sentinel converts to undefined; selecting "paid" emits "paid".
 *  - PageSizeSelector: visible and calls onPageSizeChange.
 */
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderComponent } from "@/test";
import { EnrollmentsFilterBar } from "../components/EnrollmentsFilterBar";

function renderBar(
  overrides: Partial<React.ComponentProps<typeof EnrollmentsFilterBar>> = {},
) {
  const defaults: React.ComponentProps<typeof EnrollmentsFilterBar> = {
    q: "",
    year: undefined,
    status: undefined,
    pageSize: 20,
    onQChange: vi.fn(),
    onYearChange: vi.fn(),
    onStatusChange: vi.fn(),
    onPageSizeChange: vi.fn(),
  };
  return renderComponent(<EnrollmentsFilterBar {...defaults} {...overrides} />);
}

describe("EnrollmentsFilterBar — SearchInput", () => {
  it("renders SearchInput with correct placeholder", () => {
    renderBar();
    expect(
      screen.getByPlaceholderText(/buscar por estudiante o programa/i),
    ).toBeInTheDocument();
  });

  it("calls onQChange after debounce when user types", async () => {
    const user = userEvent.setup();
    const onQChange = vi.fn();
    renderBar({ onQChange });

    const input = screen.getByPlaceholderText(
      /buscar por estudiante o programa/i,
    );
    await user.type(input, "ana");

    await waitFor(
      () => {
        expect(onQChange).toHaveBeenCalledWith("ana");
      },
      { timeout: 1000 },
    );
  });
});

describe("EnrollmentsFilterBar — Year input", () => {
  it("calls onYearChange with parsed number for valid year", async () => {
    const user = userEvent.setup();
    const onYearChange = vi.fn();
    renderBar({ onYearChange });

    const yearInput = screen.getByPlaceholderText(/año/i);
    await user.type(yearInput, "2026");

    expect(onYearChange).toHaveBeenCalledWith(2026);
  });

  it("calls onYearChange with undefined for out-of-range year", async () => {
    const user = userEvent.setup();
    const onYearChange = vi.fn();
    renderBar({ onYearChange });

    const yearInput = screen.getByPlaceholderText(/año/i);
    await user.type(yearInput, "9");

    // Partial/out-of-range → emits undefined
    const lastCall =
      onYearChange.mock.calls[onYearChange.mock.calls.length - 1];
    expect(lastCall?.[0]).toBeUndefined();
  });
});

describe("EnrollmentsFilterBar — Status Select", () => {
  it("renders Todas option (sentinel) and status options", async () => {
    const user = userEvent.setup();
    renderBar();

    await user.click(screen.getByRole("combobox", { name: /estado/i }));
    // findAllByText because "Todas" appears both in the trigger value and the option
    expect(await screen.findAllByText("Todas")).not.toHaveLength(0);
    expect(
      screen.getByRole("option", { name: "Pendiente" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "Pagada" })).toBeInTheDocument();
    expect(
      screen.getByRole("option", { name: "Cancelada" }),
    ).toBeInTheDocument();
  });

  it("selecting Todas emits undefined (not __all__)", async () => {
    const user = userEvent.setup();
    const onStatusChange = vi.fn();
    renderBar({ status: "paid", onStatusChange });

    await user.click(screen.getByRole("combobox", { name: /estado/i }));
    await user.click(await screen.findByRole("option", { name: "Todas" }));

    expect(onStatusChange).toHaveBeenCalledWith(undefined);
  });

  it("selecting Pagada emits 'paid'", async () => {
    const user = userEvent.setup();
    const onStatusChange = vi.fn();
    renderBar({ onStatusChange });

    await user.click(screen.getByRole("combobox", { name: /estado/i }));
    await user.click(await screen.findByRole("option", { name: "Pagada" }));

    expect(onStatusChange).toHaveBeenCalledWith("paid");
  });
});

describe("EnrollmentsFilterBar — PageSizeSelector", () => {
  it("renders PageSizeSelector and calls onPageSizeChange", async () => {
    const user = userEvent.setup();
    const onPageSizeChange = vi.fn();
    renderBar({ pageSize: 20, onPageSizeChange });

    const allComboboxes = screen.getAllByRole("combobox");
    // The page-size selector is the last combobox (after status select)
    const pageSizeSelect = allComboboxes[allComboboxes.length - 1];
    await user.click(pageSizeSelect);
    await user.click(await screen.findByText("50 por página"));

    expect(onPageSizeChange).toHaveBeenCalledWith(50);
  });
});
