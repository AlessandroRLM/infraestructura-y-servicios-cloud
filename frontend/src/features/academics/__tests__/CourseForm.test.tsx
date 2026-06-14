import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { CourseFormHelpers } from "../components/CourseForm";
import { CourseForm } from "../components/CourseForm";
import type { CourseFormValues } from "../schemas/course";
import { courseSchema } from "../schemas/course";

type SubmitFn = (
  values: CourseFormValues,
  helpers: CourseFormHelpers,
) => Promise<void>;

describe("CourseForm", () => {
  it("shows inline error on blur when code is empty", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn() as unknown as SubmitFn & ReturnType<typeof vi.fn>;
    render(<CourseForm onSubmit={onSubmit} />);

    const codeInput = screen.getByLabelText("Código");
    await user.click(codeInput);
    await user.tab();

    expect(
      await screen.findByText("El código es obligatorio"),
    ).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("shows inline error on blur when name is empty", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn() as unknown as SubmitFn & ReturnType<typeof vi.fn>;
    render(<CourseForm onSubmit={onSubmit} />);

    const nameInput = screen.getByLabelText("Nombre");
    await user.click(nameInput);
    await user.tab();

    expect(
      await screen.findByText("El nombre es obligatorio"),
    ).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("shows inline error when credits is not selected on submit", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn() as unknown as SubmitFn & ReturnType<typeof vi.fn>;
    render(<CourseForm onSubmit={onSubmit} />);

    await user.type(screen.getByLabelText("Código"), "CS-101");
    await user.type(screen.getByLabelText("Nombre"), "Cálculo");
    await user.click(screen.getByRole("button", { name: /guardar/i }));

    expect(
      await screen.findByText("Selecciona los créditos"),
    ).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("renders exactly 10 credit options (3–12)", async () => {
    render(<CourseForm onSubmit={vi.fn()} />);

    const trigger = screen.getByRole("combobox");
    await userEvent.click(trigger);

    const options = screen.getAllByRole("option");
    expect(options).toHaveLength(10);
    const values = options.map((o) => Number(o.textContent?.trim()));
    expect(values).toEqual([3, 4, 5, 6, 7, 8, 9, 10, 11, 12]);
  });

  it("calls onSubmit with {code, name, credits as number} when form is valid", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn() as unknown as SubmitFn & ReturnType<typeof vi.fn>;
    render(<CourseForm onSubmit={onSubmit} />);

    await user.type(screen.getByLabelText("Código"), "CS-101");
    await user.type(screen.getByLabelText("Nombre"), "Cálculo");

    const trigger = screen.getByRole("combobox");
    await user.click(trigger);
    const option5 = screen.getByRole("option", { name: "5" });
    await user.click(option5);

    await user.click(screen.getByRole("button", { name: /guardar/i }));

    await vi.waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    const [values, helpers] = onSubmit.mock.calls[0] as [
      CourseFormValues,
      CourseFormHelpers,
    ];
    expect(values).toMatchObject({
      code: "CS-101",
      name: "Cálculo",
      credits: 5,
    });
    expect(typeof values.credits).toBe("number");
    expect(typeof helpers.setError).toBe("function");
  });

  it("setError path for duplicate code renders inline error", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn(
      async (_values: CourseFormValues, helpers: CourseFormHelpers) => {
        helpers.setError("code", {
          message: "Ya existe una asignatura con ese código",
        });
      },
    ) as unknown as SubmitFn & ReturnType<typeof vi.fn>;
    render(<CourseForm onSubmit={onSubmit} />);

    await user.type(screen.getByLabelText("Código"), "DUP");
    await user.type(screen.getByLabelText("Nombre"), "Duplicado");

    const trigger = screen.getByRole("combobox");
    await user.click(trigger);
    const option = screen.getByRole("option", { name: "6" });
    await user.click(option);

    await user.click(screen.getByRole("button", { name: /guardar/i }));

    expect(
      await screen.findByText("Ya existe una asignatura con ese código"),
    ).toBeInTheDocument();
  });
});

describe("courseSchema direct", () => {
  it("SC-27: rejects out-of-set credits value", () => {
    const result = courseSchema.safeParse({ code: "X", name: "Y", credits: 2 });
    expect(result.success).toBe(false);
    if (!result.success) {
      const creditsError = result.error.issues.find((i) =>
        i.path.includes("credits"),
      );
      expect(creditsError).toBeDefined();
      expect(creditsError?.message).not.toMatch(/AlreadyExists|Code\./);
    }
  });

  it("accepts in-set credits values", () => {
    for (const c of [3, 6, 12]) {
      expect(
        courseSchema.safeParse({ code: "X", name: "Y", credits: c }).success,
      ).toBe(true);
    }
  });
});
