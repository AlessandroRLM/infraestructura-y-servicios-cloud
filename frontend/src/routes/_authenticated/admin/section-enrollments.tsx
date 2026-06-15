import { createFileRoute } from "@tanstack/react-router";
import { requireRoutePermission } from "@/features/auth";
import { SectionEnrollmentsPage } from "@/features/section-enrollments";

export const Route = createFileRoute(
  "/_authenticated/admin/section-enrollments",
)({
  beforeLoad: ({ context }) => {
    requireRoutePermission(context.session, "/admin/section-enrollments");
  },
  component: SectionEnrollmentsPage,
});
