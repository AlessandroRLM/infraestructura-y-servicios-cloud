import { createFileRoute } from "@tanstack/react-router";
import { LogoutButton } from "@/features/auth";

export const Route = createFileRoute("/_authenticated/")({
  component: Dashboard,
});

// LogoutButton lives here until a shared authenticated layout/navbar exists.
function Dashboard() {
  return (
    <div data-testid="dashboard">
      <span>Dashboard</span>
      <LogoutButton />
    </div>
  );
}
