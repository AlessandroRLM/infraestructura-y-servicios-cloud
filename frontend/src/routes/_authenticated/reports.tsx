import { createFileRoute } from "@tanstack/react-router";
import { requireAnyPermission } from "@/features/auth";
import { ReportsPage } from "@/features/reports";

export const Route = createFileRoute("/_authenticated/reports")({
  beforeLoad: ({ context }) => {
    requireAnyPermission(context.session, ["reports.read"]);
  },
  component: ReportsPage,
});
