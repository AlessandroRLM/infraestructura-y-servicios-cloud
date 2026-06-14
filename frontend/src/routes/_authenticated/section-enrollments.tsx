import { createFileRoute } from "@tanstack/react-router";
import { requireAnyPermission } from "@/features/auth";
import { SectionEnrollmentsPage } from "@/features/section-enrollments";

export const Route = createFileRoute("/_authenticated/section-enrollments")({
  beforeLoad: ({ context }) => {
    requireAnyPermission(context.session, [
      "sections.enroll",
      "section_enrollment.view_own",
    ]);
  },
  component: SectionEnrollmentsPage,
});
