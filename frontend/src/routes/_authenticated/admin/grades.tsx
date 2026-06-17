import { createFileRoute } from "@tanstack/react-router";
import { requireRoutePermission } from "@/features/auth";
import { GradesPage } from "@/features/grades";

// Admin grades route — renders the admin/teacher GradesPage.
// Scheme management is gated inside GradesPage on grades.override.
// The route-level guard accepts grades.read, grades.write, or grades.override.
// See ROUTE_PERMISSIONS["/admin/grades"].
export const Route = createFileRoute("/_authenticated/admin/grades")({
  beforeLoad: ({ context }) => {
    requireRoutePermission(context.session, "/admin/grades");
  },
  component: GradesPage,
});
