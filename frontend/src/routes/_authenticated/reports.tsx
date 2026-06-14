import { createFileRoute } from "@tanstack/react-router";
import { requireRoutePermission } from "@/features/auth";
import { ReportsPage } from "@/features/reports";

export const Route = createFileRoute("/_authenticated/reports")({
  beforeLoad: ({ context }) => {
    requireRoutePermission(context.session, "/reports");
  },
  component: ReportsPage,
});
