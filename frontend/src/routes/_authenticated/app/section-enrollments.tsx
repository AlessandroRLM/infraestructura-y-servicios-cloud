import { createFileRoute } from "@tanstack/react-router";
import { requireRoutePermission } from "@/features/auth";
import { ParticipantSectionEnrollmentsPage } from "@/features/section-enrollments";
import { ownSectionEnrollmentsSearchSchema } from "@/features/section-enrollments/schemas/search";

/**
 * Route-owned wrapper that reads search params and passes them down as props.
 * Keeps ParticipantSectionEnrollmentsPage free of any direct Route import, breaking the
 * circular dependency that would arise from app/section-enrollments.tsx ↔ the page component.
 *
 * Renders the STUDENT view: self-enrollment panel + own enrollments list.
 * The ADMIN view lives at admin/section-enrollments.tsx and is untouched.
 */
function AppSectionEnrollmentsPage() {
  const { pageSize } = Route.useSearch();
  const navigate = Route.useNavigate();

  return (
    <ParticipantSectionEnrollmentsPage
      pageSize={pageSize}
      onPageSizeChange={(n) =>
        navigate({ search: (prev) => ({ ...prev, pageSize: n }) })
      }
    />
  );
}

// Participant section-enrollments route — renders the student's own section enrollment list.
// The feature guard requires section_enrollment.view_own or sections.enroll.
// See ROUTE_PERMISSIONS["/app/section-enrollments"].
export const Route = createFileRoute("/_authenticated/app/section-enrollments")(
  {
    validateSearch: ownSectionEnrollmentsSearchSchema,
    beforeLoad: ({ context }) => {
      requireRoutePermission(context.session, "/app/section-enrollments");
    },
    component: AppSectionEnrollmentsPage,
  },
);
