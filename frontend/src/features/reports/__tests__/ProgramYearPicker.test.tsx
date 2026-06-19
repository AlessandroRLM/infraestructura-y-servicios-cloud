/**
 * ProgramYearPicker component tests.
 *
 * Covers:
 *  - Search state resets after program select and popover dismiss.
 *  - Year input uses local text state: field persists partial/invalid input mid-type
 *    and syncs when the external `year` prop changes (back/forward nav).
 */
import { create } from "@bufbuild/protobuf";
import { act, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import {
  CatalogService,
  ListProgramsResponseSchema,
  ProgramSchema,
} from "@/gen/catalog/v1/catalog_pb";
import { renderComponent } from "@/test";
import { ProgramYearPicker } from "../components/ProgramYearPicker";

function makeProgram(id: string, code: string, name: string) {
  return create(ProgramSchema, {
    id,
    code,
    name,
    createdAt: "",
    updatedAt: "",
  });
}

function makeListProgramsResponse(
  ...programs: ReturnType<typeof makeProgram>[]
) {
  return create(ListProgramsResponseSchema, { programs, nextPageToken: "" });
}

const stubTransport = makeStubTransport([
  CatalogService,
  {
    listPrograms: async () =>
      makeListProgramsResponse(
        makeProgram("prog-1", "ICI", "Ingeniería Civil"),
        makeProgram("prog-2", "IIN", "Ingeniería Industrial"),
      ),
  },
]);

describe("ProgramYearPicker — search reset", () => {
  it("CommandInput is empty after selecting a program and reopening the popover", async () => {
    const user = userEvent.setup();
    const onProgramChange = vi.fn();
    const onYearChange = vi.fn();

    renderComponent(
      <ProgramYearPicker
        programId=""
        year={undefined}
        onProgramChange={onProgramChange}
        onYearChange={onYearChange}
      />,
      { transport: stubTransport },
    );

    // Open the popover.
    const trigger = screen.getByRole("combobox");
    await user.click(trigger);

    // Wait for programs to load, then type a search query.
    await waitFor(() =>
      expect(screen.getByText("ICI — Ingeniería Civil")).toBeInTheDocument(),
    );
    const input = screen.getByPlaceholderText("Buscar programa…");
    await user.type(input, "Civil");
    expect(input).toHaveValue("Civil");

    // Select a program.
    await user.click(screen.getByText("ICI — Ingeniería Civil"));

    // Reopen the popover and verify search is reset.
    await user.click(trigger);
    await waitFor(() =>
      expect(screen.getByPlaceholderText("Buscar programa…")).toHaveValue(""),
    );
  });

  it("CommandInput is empty after dismissing the popover via Escape", async () => {
    const user = userEvent.setup();
    const onProgramChange = vi.fn();
    const onYearChange = vi.fn();

    renderComponent(
      <ProgramYearPicker
        programId=""
        year={undefined}
        onProgramChange={onProgramChange}
        onYearChange={onYearChange}
      />,
      { transport: stubTransport },
    );

    // Open the popover.
    const trigger = screen.getByRole("combobox");
    await user.click(trigger);

    // Wait for programs to load, then type a search query.
    await waitFor(() =>
      expect(screen.getByText("ICI — Ingeniería Civil")).toBeInTheDocument(),
    );
    const input = screen.getByPlaceholderText("Buscar programa…");
    await user.type(input, "Industrial");
    expect(input).toHaveValue("Industrial");

    // Dismiss via Escape.
    await user.keyboard("{Escape}");

    // Reopen the popover and verify search is reset.
    await user.click(trigger);
    await waitFor(() =>
      expect(screen.getByPlaceholderText("Buscar programa…")).toHaveValue(""),
    );
  });
});

describe("ProgramYearPicker — year input local text state", () => {
  it("typing '2026' digit-by-digit keeps the field visible and commits onYearChange(2026)", async () => {
    const user = userEvent.setup();
    const onYearChange = vi.fn();

    renderComponent(
      <ProgramYearPicker
        programId=""
        year={undefined}
        onProgramChange={vi.fn()}
        onYearChange={onYearChange}
      />,
      { transport: stubTransport },
    );

    const yearInput = screen.getByRole("spinbutton");

    await user.type(yearInput, "2026");

    // The field must show the full typed value, not reset on each keystroke.
    expect(yearInput).toHaveValue(2026);
    // onYearChange is called with 2026 once a valid year is complete.
    expect(onYearChange).toHaveBeenCalledWith(2026);
    // It must not have been called with undefined for valid input at any point.
    const lastCall = onYearChange.mock.calls.at(-1);
    expect(lastCall).toEqual([2026]);
  });

  it("partial entry keeps visible text and does not clear the field", async () => {
    const user = userEvent.setup();
    const onYearChange = vi.fn();

    renderComponent(
      <ProgramYearPicker
        programId=""
        year={undefined}
        onProgramChange={vi.fn()}
        onYearChange={onYearChange}
      />,
      { transport: stubTransport },
    );

    const yearInput = screen.getByRole("spinbutton");

    // Type only "2" — invalid/partial.
    await user.type(yearInput, "2");

    // Field must still show "2", not be empty.
    expect(yearInput).toHaveValue(2);
    // Upstream receives undefined because 2 is out of [2000,2100].
    expect(onYearChange).toHaveBeenCalledWith(undefined);
  });

  it("reflects external year prop change (back/forward nav sync)", async () => {
    const onYearChange = vi.fn();

    // A stateful wrapper inside the provider tree drives prop changes.
    let setExternalYear!: (y: number | undefined) => void;
    function ControlledWrapper() {
      const [year, setYear] = useState<number | undefined>(2024);
      setExternalYear = setYear;
      return (
        <ProgramYearPicker
          programId=""
          year={year}
          onProgramChange={vi.fn()}
          onYearChange={onYearChange}
        />
      );
    }

    renderComponent(<ControlledWrapper />, { transport: stubTransport });

    const yearInput = screen.getByRole("spinbutton");
    expect(yearInput).toHaveValue(2024);

    // Simulate external navigation updating the prop (e.g. back/forward).
    act(() => {
      setExternalYear(2025);
    });

    await waitFor(() => expect(yearInput).toHaveValue(2025));
  });

  it("editing a previously committed valid year (URL round-trip) does not blank the field", async () => {
    const user = userEvent.setup();
    const commits: (number | undefined)[] = [];

    // Wrapper that round-trips onYearChange back into the `year` prop, mirroring
    // how ReportsPage commits the value to the URL search param. This reproduces
    // the real feedback path: a keystroke that makes the value invalid commits
    // `undefined` upstream, which flows back as the `year` prop.
    function RoundTripWrapper() {
      const [year, setYear] = useState<number | undefined>(2025);
      return (
        <ProgramYearPicker
          programId=""
          year={year}
          onProgramChange={vi.fn()}
          onYearChange={(y) => {
            commits.push(y);
            setYear(y);
          }}
        />
      );
    }

    renderComponent(<RoundTripWrapper />, { transport: stubTransport });

    const yearInput = screen.getByRole("spinbutton");
    expect(yearInput).toHaveValue(2025);

    // Append a digit so the value becomes out-of-range ("20259"). The commit of
    // `undefined` must NOT wipe the visible text via the prop-sync effect.
    await user.type(yearInput, "9");

    expect(yearInput).toHaveValue(20259);
    expect(commits.at(-1)).toBe(undefined);
  });
});
