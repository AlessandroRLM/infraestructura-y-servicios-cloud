import type { ServiceImpl } from "@connectrpc/connect";
import { Code, ConnectError } from "@connectrpc/connect";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeStubTransport } from "@/core/test";
import type { AuthenticatedSession } from "@/features/auth";
import { ProfileService } from "@/gen/profiles/v1/profiles_pb";
import { renderWithProviders } from "@/test";

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

type ProfileImpl = Partial<ServiceImpl<typeof ProfileService>>;

const studentSession = {
  status: "authenticated" as const,
  userId: "user-1",
  email: "student@test.com",
  roles: ["student"],
  permissions: ["profile.view_own", "profile.edit_own"],
};

const studentSessionSource = {
  getSession: async (): Promise<AuthenticatedSession> => ({
    userId: studentSession.userId,
    email: studentSession.email,
    roles: studentSession.roles,
    permissions: studentSession.permissions,
  }),
};

function renderProfilePage(handlers: ProfileImpl = {}) {
  return renderWithProviders({
    route: "/profile",
    transport: makeStubTransport([ProfileService, handlers]),
    session: studentSession,
    sessionSource: studentSessionSource,
  });
}

describe("ProfilePage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("SC-1: renders identity card fields and pre-filled form on success", async () => {
    const { toast } = await import("sonner");

    const handlers: ProfileImpl = {
      getOwnProfile: async () => ({
        userId: "user-1",
        givenNames: "Juan Carlos",
        lastNamePaternal: "García",
        lastNameMaternal: "López",
        nationalIdType: "RUT",
        nationalId: "12345678-9",
        sex: "Masculino",
        nationality: "Chilena",
        phone: "+56912345678",
        personalEmail: "juan@gmail.com",
        birthDate: "1990-05-15",
        addressStreet: "",
        commune: "",
        region: "",
        country: "",
        postalCode: "",
        emergencyContactName: "",
        emergencyContactPhone: "",
      }),
      upsertOwnProfile: async () => ({
        userId: "user-1",
        givenNames: "Juan Carlos",
        lastNamePaternal: "García",
        nationalIdType: "RUT",
        nationalId: "12345678-9",
      }),
    };

    renderProfilePage(handlers);

    // Identity card fields visible
    expect(await screen.findByText("Juan Carlos")).toBeInTheDocument();
    expect(screen.getByText("García")).toBeInTheDocument();
    expect(screen.getByText("López")).toBeInTheDocument();
    expect(screen.getByText("RUT")).toBeInTheDocument();
    expect(screen.getByText("12345678-9")).toBeInTheDocument();
    expect(screen.getByText("Masculino")).toBeInTheDocument();
    expect(screen.getByText("Chilena")).toBeInTheDocument();

    // Form is pre-populated — use id selector to avoid ambiguity
    const phoneInput = document.getElementById(
      "profile-phone",
    ) as HTMLInputElement;
    expect(phoneInput).not.toBeNull();
    expect(phoneInput.value).toBe("+56912345678");

    expect(toast.error).not.toHaveBeenCalled();
  });

  it("SC-2: shows NotFound callout and empty form, no error toast", async () => {
    const { toast } = await import("sonner");

    const handlers: ProfileImpl = {
      getOwnProfile: async () => {
        throw new ConnectError("not found", Code.NotFound);
      },
    };

    renderProfilePage(handlers);

    expect(await screen.findByText("Completa tu perfil")).toBeInTheDocument();
    expect(
      screen.getByText(/agrega tus datos de contacto/i),
    ).toBeInTheDocument();

    // Form rendered empty — use id selector to avoid ambiguity with two "Teléfono" labels
    const phoneInput = document.getElementById(
      "profile-phone",
    ) as HTMLInputElement;
    expect(phoneInput).not.toBeNull();
    expect(phoneInput.value).toBe("");

    // No identity card
    expect(screen.queryByText("Datos de identidad")).not.toBeInTheDocument();

    expect(toast.error).not.toHaveBeenCalled();
  });

  it("SC-3: save success calls upsertOwnProfile, shows toast.success", async () => {
    const user = userEvent.setup();
    const { toast } = await import("sonner");

    let upsertCalledWith: Record<string, unknown> | null = null;

    const handlers: ProfileImpl = {
      getOwnProfile: async () => ({
        userId: "user-1",
        givenNames: "Juan",
        lastNamePaternal: "García",
        nationalIdType: "RUT",
        nationalId: "12345678-9",
        phone: "+56900000000",
      }),
      upsertOwnProfile: async (req) => {
        upsertCalledWith = { ...req };
        return {
          userId: "user-1",
          givenNames: "Juan",
          lastNamePaternal: "García",
          nationalIdType: "RUT",
          nationalId: "12345678-9",
          phone: req.phone,
        };
      },
    };

    renderProfilePage(handlers);

    await screen.findByText("Juan");

    const phoneInput = document.getElementById(
      "profile-phone",
    ) as HTMLInputElement;
    await user.clear(phoneInput);
    await user.type(phoneInput, "+56912345678");

    await user.click(screen.getByRole("button", { name: /guardar/i }));

    await waitFor(() => {
      expect(toast.success).toHaveBeenCalledWith("Perfil actualizado");
    });

    expect(upsertCalledWith).not.toBeNull();
  });

  it("SC-4: transport error on save shows toast.error, no field error", async () => {
    const user = userEvent.setup();
    const { toast } = await import("sonner");

    const handlers: ProfileImpl = {
      getOwnProfile: async () => ({
        userId: "user-1",
        givenNames: "Juan",
        lastNamePaternal: "García",
        nationalIdType: "RUT",
        nationalId: "12345678-9",
      }),
      upsertOwnProfile: async () => {
        throw new ConnectError("internal server error", Code.Internal);
      },
    };

    renderProfilePage(handlers);

    await screen.findByText("Juan");

    await user.click(screen.getByRole("button", { name: /guardar/i }));

    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith("Error al guardar el perfil");
    });

    // No inline birthDate error shown
    expect(
      screen.queryByText("Fecha inválida (formato AAAA-MM-DD)"),
    ).not.toBeInTheDocument();
  });

  it("WARNING-1: clearing a pre-filled field sends empty string (not undefined)", async () => {
    const user = userEvent.setup();

    let capturedPhone: string | undefined = "NOT_CALLED";

    const handlers: ProfileImpl = {
      getOwnProfile: async () => ({
        userId: "user-1",
        givenNames: "Ana",
        lastNamePaternal: "Pérez",
        nationalIdType: "RUT",
        nationalId: "11111111-1",
        phone: "+56911111111",
      }),
      upsertOwnProfile: async (req) => {
        capturedPhone = req.phone;
        return {
          userId: "user-1",
          givenNames: "Ana",
          lastNamePaternal: "Pérez",
          nationalIdType: "RUT",
          nationalId: "11111111-1",
          phone: req.phone,
        };
      },
    };

    renderProfilePage(handlers);

    await screen.findByText("Ana");

    const phoneInput = document.getElementById(
      "profile-phone",
    ) as HTMLInputElement;
    expect(phoneInput.value).toBe("+56911111111");

    // Clear the phone field entirely
    await user.clear(phoneInput);
    expect(phoneInput.value).toBe("");

    await user.click(screen.getByRole("button", { name: /guardar/i }));

    await waitFor(() => {
      // Present-empty "" must reach the backend — NOT undefined
      expect(capturedPhone).toBe("");
    });
  });

  it("WARNING-2: InvalidArgument error shows inline birthDate error, no toast", async () => {
    const user = userEvent.setup();
    const { toast } = await import("sonner");

    const handlers: ProfileImpl = {
      getOwnProfile: async () => ({
        userId: "user-1",
        givenNames: "Pedro",
        lastNamePaternal: "Soto",
        nationalIdType: "RUT",
        nationalId: "22222222-2",
      }),
      upsertOwnProfile: async () => {
        throw new ConnectError("invalid argument", Code.InvalidArgument);
      },
    };

    renderProfilePage(handlers);

    await screen.findByText("Pedro");

    await user.click(screen.getByRole("button", { name: /guardar/i }));

    await waitFor(() => {
      expect(
        screen.getByText("Fecha inválida (formato AAAA-MM-DD)"),
      ).toBeInTheDocument();
    });

    expect(toast.error).not.toHaveBeenCalled();
    expect(toast.success).not.toHaveBeenCalled();
  });

  it("WARNING-3: renders loading skeleton while GetOwnProfile is pending", async () => {
    // Never-resolving handler keeps the query in the pending/loading state
    const handlers: ProfileImpl = {
      // biome-ignore lint/suspicious/noEmptyBlockStatements: intentional never-resolving promise for loading state test
      getOwnProfile: () => new Promise(() => {}),
    };

    renderProfilePage(handlers);

    // TanStack Router renders asynchronously — wait for the page to mount.
    // In loading state ProfilePage renders Skeletons immediately (no h1).
    // The Skeleton component stamps data-slot="skeleton" on every element.
    await waitFor(() => {
      const skeletonEls = document.querySelectorAll('[data-slot="skeleton"]');
      expect(skeletonEls.length).toBeGreaterThan(0);
    });

    // The form and identity card must NOT be visible while loading
    expect(document.getElementById("profile-phone")).not.toBeInTheDocument();
  });
});
