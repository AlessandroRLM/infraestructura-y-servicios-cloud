import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { ProgramFormHelpers } from "../components/ProgramForm";
import { ProgramForm } from "../components/ProgramForm";
import type { ProgramFormValues } from "../schemas/program";

type SubmitFn = (
  values: ProgramFormValues,
  helpers: ProgramFormHelpers,
) => Promise<void>;

describe("ProgramForm", () => {
  it("shows inline errors on blur when fields are empty, and does not call onSubmit", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn() as unknown as SubmitFn & ReturnType<typeof vi.fn>;
    render(<ProgramForm onSubmit={onSubmit} />);

    const codeInput = screen.getByLabelText("Código");
    const nameInput = screen.getByLabelText("Nombre");

    await user.click(codeInput);
    await user.tab();
    expect(
      await screen.findByText("El código es obligatorio"),
    ).toBeInTheDocument();

    await user.click(nameInput);
    await user.tab();
    expect(
      await screen.findByText("El nombre es obligatorio"),
    ).toBeInTheDocument();

    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("calls onSubmit with {code, name} and helpers when form is valid", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn() as unknown as SubmitFn & ReturnType<typeof vi.fn>;
    render(<ProgramForm onSubmit={onSubmit} />);

    await user.type(screen.getByLabelText("Código"), "ING-01");
    await user.type(screen.getByLabelText("Nombre"), "Ingeniería de Software");
    await user.click(screen.getByRole("button", { name: /guardar/i }));

    await vi.waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    const [values, helpers] = onSubmit.mock.calls[0] as [
      ProgramFormValues,
      ProgramFormHelpers,
    ];
    expect(values).toMatchObject({
      code: "ING-01",
      name: "Ingeniería de Software",
    });
    expect(typeof helpers.setError).toBe("function");
  });
});
