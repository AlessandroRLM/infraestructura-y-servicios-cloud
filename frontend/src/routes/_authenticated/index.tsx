import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/")({
  component: () => <div data-testid="dashboard">Dashboard</div>,
});
