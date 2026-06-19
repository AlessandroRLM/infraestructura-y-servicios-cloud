import { createFileRoute } from "@tanstack/react-router";
import { requireRoutePermission } from "@/features/auth";
import { ReportsPage } from "@/features/reports";
import { validateSearch } from "@/features/reports/schemas/reportsSearch";

export const Route = createFileRoute("/_authenticated/admin/reports")({
  validateSearch: validateSearch,
  beforeLoad: ({ context }) => {
    requireRoutePermission(context.session, "/admin/reports");
  },
  component: ReportsPage,
});
