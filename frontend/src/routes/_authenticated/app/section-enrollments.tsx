import { createFileRoute } from "@tanstack/react-router";
import { requireRoutePermission } from "@/features/auth";
import { SectionEnrollmentsPage } from "@/features/section-enrollments";

export const Route = createFileRoute("/_authenticated/app/section-enrollments")(
  {
    beforeLoad: ({ context }) => {
      requireRoutePermission(context.session, "/app/section-enrollments");
    },
    component: SectionEnrollmentsPage,
  },
);
