/**
 * ProgramPicker component tests.
 *
 * Covers:
 *  - Lists programs from the stubbed listPrograms RPC.
 *  - Selecting a program emits the programId and shows the label.
 *  - Search input is cleared after popover closes or program is selected.
 */
import { create } from "@bufbuild/protobuf";
import { screen, waitFor } from "@testing-library/react";
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
import { ProgramPicker } from "../components/ProgramPicker";

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

describe("ProgramPicker — lists programs", () => {
  it("opens popover and shows programs from the stub", async () => {
    const user = userEvent.setup();
    renderComponent(<ProgramPicker value="" onChange={vi.fn()} />, {
      transport: stubTransport,
    });

    const trigger = screen.getByRole("combobox");
    await user.click(trigger);

    await waitFor(() =>
      expect(screen.getByText("ICI — Ingeniería Civil")).toBeInTheDocument(),
    );
    expect(screen.getByText("IIN — Ingeniería Industrial")).toBeInTheDocument();
  });

  it("selecting a program calls onChange with programId and caches label via selectedLabel", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();

    // Use a stateful wrapper so that value prop updates after selection,
    // allowing the selectedLabel cache to render correctly on the trigger.
    function Wrapper() {
      const [value, setValue] = useState("");
      return (
        <ProgramPicker
          value={value}
          onChange={(id) => {
            setValue(id);
            onChange(id);
          }}
        />
      );
    }

    renderComponent(<Wrapper />, { transport: stubTransport });

    const trigger = screen.getByRole("combobox");
    await user.click(trigger);

    await waitFor(() =>
      expect(screen.getByText("ICI — Ingeniería Civil")).toBeInTheDocument(),
    );

    await user.click(screen.getByText("ICI — Ingeniería Civil"));

    expect(onChange).toHaveBeenCalledWith("prog-1");
    // After selection with updated value prop, trigger shows the cached label.
    await waitFor(() =>
      expect(trigger).toHaveTextContent("ICI — Ingeniería Civil"),
    );
  });

  it("search text is cleared after selecting a program (state reset)", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();

    renderComponent(<ProgramPicker value="" onChange={onChange} />, {
      transport: stubTransport,
    });

    const trigger = screen.getByRole("combobox");
    await user.click(trigger);

    await waitFor(() =>
      expect(screen.getByText("ICI — Ingeniería Civil")).toBeInTheDocument(),
    );

    const searchInput = screen.getByPlaceholderText("Buscar programa…");
    await user.type(searchInput, "Civil");
    expect(searchInput).toHaveValue("Civil");

    await user.click(screen.getByText("ICI — Ingeniería Civil"));

    // After selecting, onChange should have been called.
    await waitFor(() => expect(onChange).toHaveBeenCalledWith("prog-1"));

    // Reopen and verify search input is cleared.
    await user.click(trigger);
    await waitFor(() =>
      expect(screen.getByPlaceholderText("Buscar programa…")).toHaveValue(""),
    );
  });
});
