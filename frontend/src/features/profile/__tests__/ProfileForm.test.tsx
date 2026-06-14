import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { ProfileFormHelpers } from "../components/ProfileForm";
import { ProfileForm } from "../components/ProfileForm";
import type { ProfileFormValues } from "../schemas/profile";

type SubmitFn = (
  values: ProfileFormValues,
  helpers: ProfileFormHelpers,
) => Promise<void>;

describe("ProfileForm", () => {
  it("shows inline error on blur when personalEmail is invalid, and does not call onSubmit", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn() as unknown as SubmitFn & ReturnType<typeof vi.fn>;
    render(<ProfileForm onSubmit={onSubmit} />);

    const emailInput = screen.getByLabelText("Correo personal");

    await user.click(emailInput);
    await user.type(emailInput, "not-an-email");
    await user.tab();

    expect(await screen.findByText("Email inválido")).toBeInTheDocument();

    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("calls onSubmit with all 10 keys and helpers when form is valid", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn() as unknown as SubmitFn & ReturnType<typeof vi.fn>;
    render(<ProfileForm onSubmit={onSubmit} />);

    // Use id-based queries to avoid ambiguity between the two phone fields
    const phoneInput = document.getElementById(
      "profile-phone",
    ) as HTMLInputElement;
    const emergencyPhoneInput = document.getElementById(
      "profile-emergencyContactPhone",
    ) as HTMLInputElement;

    await user.type(phoneInput, "+56912345678");
    await user.type(
      screen.getByLabelText("Correo personal"),
      "test@example.com",
    );
    await user.type(screen.getByLabelText("Calle y número"), "Av. Test 123");
    await user.type(screen.getByLabelText("Comuna"), "Santiago");
    await user.type(screen.getByLabelText("Región"), "Metropolitana");
    await user.type(screen.getByLabelText("País"), "Chile");
    await user.type(screen.getByLabelText("Código postal"), "8320000");
    await user.type(screen.getByLabelText("Nombre"), "María González");
    await user.type(emergencyPhoneInput, "+56987654321");

    await user.click(screen.getByRole("button", { name: /guardar/i }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));

    const [values, helpers] = onSubmit.mock.calls[0] as [
      ProfileFormValues,
      ProfileFormHelpers,
    ];

    expect(Object.keys(values)).toHaveLength(10);
    expect(values.phone).toBe("+56912345678");
    expect(values.personalEmail).toBe("test@example.com");
    expect(typeof helpers.setError).toBe("function");
  });

  it("DatePicker: selecting a day emits a YYYY-MM-DD string (timezone-safe)", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn() as unknown as SubmitFn & ReturnType<typeof vi.fn>;
    render(<ProfileForm onSubmit={onSubmit} />);

    // Open the date popover — accessible name comes from the linked <Label>
    const dateButton = screen.getByRole("button", {
      name: /fecha de nacimiento/i,
    });
    await user.click(dateButton);

    // Wait for calendar to open — find a day button that shows "15"
    // DayPicker renders each day as a button; its text content is the day number
    const day15 = await screen.findByText("15", { selector: "button" });
    await user.click(day15);

    // Submit the form
    await user.click(screen.getByRole("button", { name: /guardar/i }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));

    const [values] = onSubmit.mock.calls[0] as [
      ProfileFormValues,
      ProfileFormHelpers,
    ];

    // birthDate must be a valid YYYY-MM-DD string with day 15
    expect(values.birthDate).toMatch(/^\d{4}-\d{2}-15$/);
  });
});
