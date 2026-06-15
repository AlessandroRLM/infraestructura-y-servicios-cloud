import { createFileRoute } from "@tanstack/react-router";
import { requireAnyPermission } from "@/features/auth";
import { SectionEnrollmentsPage } from "@/features/section-enrollments";

// Migration shim: flat route accepts both admin and participant section-enrollment
// permissions until WU-5 (Slice 2) creates the /app/section-enrollments route.
// Delete this file in WU-8 once the new area routes are live.
export const Route = createFileRoute("/_authenticated/section-enrollments")({
  beforeLoad: ({ context }) => {
    requireAnyPermission(context.session, [
      "enrollment.manage",
      "section_enrollment.view_own",
      "sections.enroll",
    ]);
  },
  component: SectionEnrollmentsPage,
});
