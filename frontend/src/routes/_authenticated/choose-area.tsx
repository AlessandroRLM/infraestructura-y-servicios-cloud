import { createFileRoute, redirect, useNavigate } from "@tanstack/react-router";
import { GraduationCap, Settings2 } from "lucide-react";
import { writePreferredArea } from "@/features/auth";

// Single-eligible users never see this screen — beforeLoad redirects them.
// Zero-eligibility users are sent to /forbidden. Only dual-eligible users land
// here and can select their preferred area.
export const Route = createFileRoute("/_authenticated/choose-area")({
  beforeLoad: ({ context }) => {
    const { eligibility } = context;
    if (eligibility.length === 0) {
      throw redirect({ to: "/forbidden" });
    }
    if (eligibility.length === 1) {
      // Single-eligible: send directly to the only available area.
      if (eligibility.includes("admin")) {
        throw redirect({ to: "/admin" });
      }
      throw redirect({ to: "/app" });
    }
    // Dual-eligible: fall through and render the chooser.
  },
  component: ChooseAreaPage,
});

function ChooseAreaPage() {
  const navigate = useNavigate();

  function selectAdmin() {
    writePreferredArea("admin");
    void navigate({ to: "/admin" });
  }

  function selectParticipant() {
    writePreferredArea("participant");
    void navigate({ to: "/app" });
  }

  return (
    <div className="flex min-h-svh items-center justify-center bg-background p-6">
      <div className="w-full max-w-lg space-y-8 text-center">
        <div className="space-y-2">
          <h1 className="text-2xl font-semibold tracking-tight">
            ¿Con qué perfil quieres entrar?
          </h1>
          <p className="text-muted-foreground text-sm">
            Tu cuenta tiene acceso a más de un área. Selecciona con cuál quieres
            continuar.
          </p>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <button
            type="button"
            onClick={selectAdmin}
            className="flex flex-col items-center gap-4 rounded-xl border bg-card p-8 text-card-foreground shadow-sm transition-colors hover:border-primary hover:bg-accent"
          >
            <span className="flex size-14 items-center justify-center rounded-full bg-primary/10">
              <Settings2 className="size-7 text-primary" aria-hidden />
            </span>
            <div className="space-y-1">
              <div className="font-semibold">Entrar como Administrador</div>
              <div className="text-muted-foreground text-xs">
                Gestión institucional
              </div>
            </div>
          </button>

          <button
            type="button"
            onClick={selectParticipant}
            className="flex flex-col items-center gap-4 rounded-xl border bg-card p-8 text-card-foreground shadow-sm transition-colors hover:border-primary hover:bg-accent"
          >
            <span className="flex size-14 items-center justify-center rounded-full bg-primary/10">
              <GraduationCap className="size-7 text-primary" aria-hidden />
            </span>
            <div className="space-y-1">
              <div className="font-semibold">
                Entrar como Docente/Estudiante
              </div>
              <div className="text-muted-foreground text-xs">
                Docente o estudiante
              </div>
            </div>
          </button>
        </div>
      </div>
    </div>
  );
}
