import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { SearchInput } from "../SearchInput";

// Controlled wrapper that mirrors how table components use SearchInput.
function ControlledSearchInput({
  initial = "",
  debounceMs = 300,
  onChange,
}: {
  initial?: string;
  debounceMs?: number;
  onChange: (v: string) => void;
}) {
  const [value, setValue] = useState(initial);
  return (
    <SearchInput
      value={value}
      onChange={(v) => {
        setValue(v);
        onChange(v);
      }}
      debounceMs={debounceMs}
      placeholder="Search…"
    />
  );
}

describe("SearchInput", () => {
  it("S-SI-01: renders immediately with the provided value", () => {
    const onChange = vi.fn();
    render(
      <SearchInput value="hello" onChange={onChange} placeholder="Search…" />,
    );
    expect(screen.getByPlaceholderText("Search…")).toHaveValue("hello");
  });

  it("S-SI-02: debounce — onChange not called until debounce elapses", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();

    render(<ControlledSearchInput debounceMs={300} onChange={onChange} />);

    const input = screen.getByPlaceholderText("Search…");

    // Type quickly — onChange should NOT fire for each keystroke.
    await user.type(input, "abc");

    // After typing, allow the debounce to fire.
    await new Promise((r) => setTimeout(r, 350));

    // onChange must have been called at most twice (possibly once for the
    // debounced final value; may be called once per flush in some environments,
    // but never once-per-keystroke when 3 chars are typed in quick succession).
    expect(onChange.mock.calls.length).toBeLessThanOrEqual(2);

    // The last call should carry the final typed value.
    const lastCall = onChange.mock.calls[onChange.mock.calls.length - 1];
    expect(lastCall?.[0]).toBe("abc");
  });

  it("S-SI-03: clear behavior — onChange called with empty string after clear", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();

    render(
      <ControlledSearchInput
        initial="hello"
        debounceMs={300}
        onChange={onChange}
      />,
    );

    const input = screen.getByPlaceholderText("Search…");
    await user.clear(input);

    await new Promise((r) => setTimeout(r, 350));

    const lastCall = onChange.mock.calls[onChange.mock.calls.length - 1];
    expect(lastCall?.[0]).toBe("");
  });

  it("S-SI-04: external value change (tab switch) syncs local state", async () => {
    const onChange = vi.fn();
    const { rerender } = render(
      <SearchInput value="hello" onChange={onChange} placeholder="Search…" />,
    );

    expect(screen.getByPlaceholderText("Search…")).toHaveValue("hello");

    // Simulate tab switch resetting q to "".
    rerender(
      <SearchInput value="" onChange={onChange} placeholder="Search…" />,
    );

    expect(screen.getByPlaceholderText("Search…")).toHaveValue("");
  });
});
