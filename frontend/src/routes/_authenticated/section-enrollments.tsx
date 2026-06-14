import { createFileRoute } from "@tanstack/react-router";
import { requireRoutePermission } from "@/features/auth";
import { SectionEnrollmentsPage } from "@/features/section-enrollments";

export const Route = createFileRoute("/_authenticated/section-enrollments")({
  beforeLoad: ({ context }) => {
    requireRoutePermission(context.session, "/section-enrollments");
  },
  component: SectionEnrollmentsPage,
});
