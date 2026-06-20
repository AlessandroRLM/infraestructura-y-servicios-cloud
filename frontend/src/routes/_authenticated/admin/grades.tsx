import { createFileRoute, Outlet } from "@tanstack/react-router";
import { requireRoutePermission } from "@/features/auth";

// Grades layout route — applies the feature permission guard for all /admin/grades/*
// routes (index + $sectionId). Renders an <Outlet /> so the matched child (the section
// selection table or the recording grid) fills this slot.
// See ROUTE_PERMISSIONS["/admin/grades"].
export const Route = createFileRoute("/_authenticated/admin/grades")({
  beforeLoad: ({ context }) => {
    requireRoutePermission(context.session, "/admin/grades");
  },
  component: () => <Outlet />,
});
