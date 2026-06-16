import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderComponent } from "@/test";
import { EvaluationSchemeForm } from "../components/EvaluationSchemeForm";

/**
 * Helper: renders EvaluationSchemeForm with sensible defaults.
 * Callers override only what they care about for each test.
 */
function renderForm(
  overrides: Partial<React.ComponentProps<typeof EvaluationSchemeForm>> = {},
) {
  const onSubmit = vi.fn();
  const props = {
    mode: "create" as const,
    initialRows: [],
    onSubmit,
    isSubmitting: false,
    schemeStateKnown: true,
    ...overrides,
  };
  renderComponent(<EvaluationSchemeForm {...props} />);
  return { onSubmit };
}

// ──────────────────────────────────────────────
// Submit gate — total !== 100 blocks
// ──────────────────────────────────────────────

describe("EvaluationSchemeForm — submit gate", () => {
  it("submit button is disabled when total is 0 (empty default)", () => {
    renderForm();
    expect(
      screen.getByRole("button", { name: /crear esquema/i }),
    ).toBeDisabled();
  });

  it("submit button is enabled only when total equals exactly 100", async () => {
    const user = userEvent.setup();
    renderForm({ initialRows: [{ percent: 30 }, { percent: 70 }] });

    const submitBtn = screen.getByRole("button", { name: /crear esquema/i });
    expect(submitBtn).not.toBeDisabled();

    // Change first row to 40 → total becomes 110 → disabled.
    const inputs = screen.getAllByRole("spinbutton");
    await user.clear(inputs[0]);
    await user.type(inputs[0], "40");
    await user.tab(); // blur

    expect(submitBtn).toBeDisabled();
  });

  it("submit button is disabled when schemeStateKnown=false even if total=100", () => {
    renderForm({
      initialRows: [{ percent: 100 }],
      schemeStateKnown: false,
    });
    expect(
      screen.getByRole("button", { name: /crear esquema/i }),
    ).toBeDisabled();
  });

  it("submit button is disabled when isSubmitting=true", () => {
    renderForm({
      initialRows: [{ percent: 100 }],
      isSubmitting: true,
    });
    expect(screen.getByRole("button", { name: /guardando/i })).toBeDisabled();
  });
});

// ──────────────────────────────────────────────
// Add / remove row
// ──────────────────────────────────────────────

describe("EvaluationSchemeForm — add/remove rows", () => {
  it("adds a row when Agregar evaluación is clicked", async () => {
    const user = userEvent.setup();
    renderForm({ initialRows: [{ percent: 50 }] });

    expect(screen.queryAllByRole("spinbutton")).toHaveLength(1);

    await user.click(
      screen.getByRole("button", { name: /agregar evaluación/i }),
    );

    expect(screen.getAllByRole("spinbutton")).toHaveLength(2);
    expect(screen.getByText("Evaluación 2")).toBeInTheDocument();
  });

  it("removes a row when delete button is clicked", async () => {
    const user = userEvent.setup();
    renderForm({ initialRows: [{ percent: 30 }, { percent: 70 }] });

    expect(screen.getAllByRole("spinbutton")).toHaveLength(2);

    await user.click(
      screen.getByRole("button", { name: /eliminar evaluación 2/i }),
    );

    expect(screen.getAllByRole("spinbutton")).toHaveLength(1);
    expect(screen.queryByText("Evaluación 2")).not.toBeInTheDocument();
  });

  it("delete button is disabled when only one row remains", () => {
    renderForm({ initialRows: [{ percent: 100 }] });
    expect(
      screen.getByRole("button", { name: /eliminar evaluación 1/i }),
    ).toBeDisabled();
  });
});

// ──────────────────────────────────────────────
// Blur validation
// ──────────────────────────────────────────────

describe("EvaluationSchemeForm — blur validation", () => {
  it("shows error on blur when field is empty", async () => {
    const user = userEvent.setup();
    renderForm();

    const input = screen.getByRole("spinbutton");
    await user.click(input);
    await user.tab(); // blur without entering a value

    await screen.findByRole("alert");
  });

  it("shows 'Usa números enteros' on blur with decimal value", async () => {
    const user = userEvent.setup();
    renderForm();

    const input = screen.getByRole("spinbutton");
    await user.clear(input);
    await user.type(input, "33.5");
    await user.tab(); // blur

    await screen.findByText(/usa números enteros/i);
  });
});

// ──────────────────────────────────────────────
// Exact submit payload
// ──────────────────────────────────────────────

describe("EvaluationSchemeForm — submit payload", () => {
  it("calls onSubmit with ['0.300','0.300','0.400'] for rows [30,30,40]", async () => {
    const user = userEvent.setup();
    const { onSubmit } = renderForm({
      initialRows: [{ percent: 30 }, { percent: 30 }, { percent: 40 }],
    });

    await user.click(screen.getByRole("button", { name: /crear esquema/i }));

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith(["0.300", "0.300", "0.400"]);
    });
  });

  it("calls onSubmit with ['1.000'] for a single row at 100%", async () => {
    const user = userEvent.setup();
    const { onSubmit } = renderForm({
      initialRows: [{ percent: 100 }],
    });

    await user.click(screen.getByRole("button", { name: /crear esquema/i }));

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith(["1.000"]);
    });
  });
});

// ──────────────────────────────────────────────
// Error display
// ──────────────────────────────────────────────

describe("EvaluationSchemeForm — error display", () => {
  it("shows FailedPrecondition message when submitError='precondition'", () => {
    renderForm({
      initialRows: [{ percent: 100 }],
      submitError: "precondition",
    });
    expect(
      screen.getByText(/este curso ya tiene notas registradas/i),
    ).toBeInTheDocument();
  });

  it("shows AlreadyExists message when submitError='already-exists'", () => {
    renderForm({
      initialRows: [{ percent: 100 }],
      submitError: "already-exists",
    });
    expect(screen.getByText(/el esquema ya existe/i)).toBeInTheDocument();
  });

  it("shows no error when submitError is null", () => {
    renderForm({
      initialRows: [{ percent: 100 }],
      submitError: null,
    });
    expect(
      screen.queryByText(/este curso ya tiene notas registradas/i),
    ).not.toBeInTheDocument();
    expect(screen.queryByText(/el esquema ya existe/i)).not.toBeInTheDocument();
  });
});

// ──────────────────────────────────────────────
// Recreate mode — AlertDialog confirm
// ──────────────────────────────────────────────

describe("EvaluationSchemeForm — recreate mode", () => {
  it("shows AlertDialog on submit in recreate mode", async () => {
    const user = userEvent.setup();
    const { onSubmit } = renderForm({
      mode: "recreate",
      initialRows: [{ percent: 100 }],
    });

    await user.click(screen.getByRole("button", { name: /recrear esquema/i }));

    // AlertDialog should be visible.
    await screen.findByRole("alertdialog");
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("calls onSubmit after confirming in recreate mode", async () => {
    const user = userEvent.setup();
    const { onSubmit } = renderForm({
      mode: "recreate",
      initialRows: [{ percent: 100 }],
    });

    await user.click(screen.getByRole("button", { name: /recrear esquema/i }));
    await screen.findByRole("alertdialog");

    await user.click(screen.getByRole("button", { name: /^recrear$/i }));

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith(["1.000"]);
    });
  });

  it("does NOT call onSubmit when cancel is clicked in AlertDialog", async () => {
    const user = userEvent.setup();
    const { onSubmit } = renderForm({
      mode: "recreate",
      initialRows: [{ percent: 100 }],
    });

    await user.click(screen.getByRole("button", { name: /recrear esquema/i }));
    await screen.findByRole("alertdialog");

    await user.click(screen.getByRole("button", { name: /cancelar/i }));

    expect(onSubmit).not.toHaveBeenCalled();
  });
});
