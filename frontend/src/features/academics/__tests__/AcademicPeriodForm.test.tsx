import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { AcademicPeriodFormHelpers } from "../components/AcademicPeriodForm";
import { AcademicPeriodForm } from "../components/AcademicPeriodForm";
import type { AcademicPeriodFormValues } from "../schemas/academicPeriod";
import { academicPeriodSchema } from "../schemas/academicPeriod";

type SubmitFn = (
  values: AcademicPeriodFormValues,
  helpers: AcademicPeriodFormHelpers,
) => Promise<void>;

describe("AcademicPeriodForm", () => {
  it("shows inline error on blur when year is empty", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn() as unknown as SubmitFn & ReturnType<typeof vi.fn>;
    render(<AcademicPeriodForm onSubmit={onSubmit} />);

    const yearInput = screen.getByLabelText("Año");
    await user.click(yearInput);
    await user.tab();

    expect(
      await screen.findByText(/El año es obligatorio/),
    ).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("shows inline error on blur when term is empty", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn() as unknown as SubmitFn & ReturnType<typeof vi.fn>;
    render(<AcademicPeriodForm onSubmit={onSubmit} />);

    const termInput = screen.getByLabelText("Semestre");
    await user.click(termInput);
    await user.tab();

    expect(
      await screen.findByText(/El semestre es obligatorio/),
    ).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("calls onSubmit with correct types when form is valid", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn() as unknown as SubmitFn & ReturnType<typeof vi.fn>;
    render(<AcademicPeriodForm onSubmit={onSubmit} />);

    await user.clear(screen.getByLabelText("Año"));
    await user.type(screen.getByLabelText("Año"), "2026");
    await user.clear(screen.getByLabelText("Semestre"));
    await user.type(screen.getByLabelText("Semestre"), "1");
    await user.type(screen.getByLabelText("Inicio"), "2026-03-01");
    await user.type(screen.getByLabelText("Término"), "2026-07-15");

    await user.click(screen.getByRole("button", { name: /guardar/i }));

    await vi.waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    const [values] = onSubmit.mock.calls[0] as [
      AcademicPeriodFormValues,
      AcademicPeriodFormHelpers,
    ];
    expect(values.year).toBe(2026);
    expect(typeof values.year).toBe("number");
    expect(values.term).toBe(1);
    expect(typeof values.term).toBe("number");
    expect(values.startDate).toBe("2026-03-01");
    expect(values.endDate).toBe("2026-07-15");
  });
});

describe("academicPeriodSchema direct", () => {
  it("rejects term outside 1-2", () => {
    const result = academicPeriodSchema.safeParse({
      year: 2025,
      term: 3,
      startDate: "2025-03-01",
      endDate: "2025-07-15",
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      const termError = result.error.issues.find((i) =>
        i.path.includes("term"),
      );
      expect(termError).toBeDefined();
    }
  });

  it("rejects endDate before startDate", () => {
    const result = academicPeriodSchema.safeParse({
      year: 2025,
      term: 1,
      startDate: "2025-07-15",
      endDate: "2025-03-01",
    });
    expect(result.success).toBe(false);
    if (!result.success) {
      const endError = result.error.issues.find((i) =>
        i.path.includes("endDate"),
      );
      expect(endError).toBeDefined();
    }
  });

  it("accepts valid period data", () => {
    expect(
      academicPeriodSchema.safeParse({
        year: 2025,
        term: 1,
        startDate: "2025-03-01",
        endDate: "2025-07-15",
      }).success,
    ).toBe(true);
  });
});
