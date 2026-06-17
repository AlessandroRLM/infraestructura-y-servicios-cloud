import { createFileRoute } from "@tanstack/react-router";
import { requireRoutePermission } from "@/features/auth";
import { GradesPage } from "@/features/grades";

// Admin grades route — renders the admin/teacher GradesPage.
// The route-level guard accepts grades.read or grades.write (teacher overlap).
// Scheme management is gated inside GradesPage on grades.override, which the
// RBAC seed always co-assigns with grades.read/write, so override holders
// always clear this guard. See ROUTE_PERMISSIONS["/admin/grades"].
export const Route = createFileRoute("/_authenticated/admin/grades")({
  beforeLoad: ({ context }) => {
    requireRoutePermission(context.session, "/admin/grades");
  },
  component: GradesPage,
});
