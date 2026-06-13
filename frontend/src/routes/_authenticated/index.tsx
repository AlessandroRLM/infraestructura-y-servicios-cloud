import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/")({
  component: Dashboard,
});

function Dashboard() {
  return (
    <div className="space-y-1" data-testid="dashboard">
      <h1 className="font-semibold text-2xl tracking-tight">Inicio</h1>
      <p className="text-muted-foreground">Bienvenido de nuevo.</p>
    </div>
  );
}
