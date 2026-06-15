import { createFileRoute } from "@tanstack/react-router";
import { requireRoutePermission } from "@/features/auth";
import { EnrollmentsPage } from "@/features/enrollments";

export const Route = createFileRoute("/_authenticated/enrollments")({
  beforeLoad: ({ context }) => {
    requireRoutePermission(context.session, "/admin/enrollments");
  },
  component: EnrollmentsPage,
});
