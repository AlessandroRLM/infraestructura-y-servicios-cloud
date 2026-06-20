import { createFileRoute } from "@tanstack/react-router";
import { requireRoutePermission } from "@/features/auth";
import { OwnEnrollmentsList } from "@/features/enrollments/components/OwnEnrollmentsList";
import { ownEnrollmentsSearchSchema } from "@/features/enrollments";

/**
 * Route-owned wrapper that reads search params and passes them down as props.
 * Keeps OwnEnrollmentsList free of any direct Route import, breaking the
 * circular dependency that would arise from app/enrollments.tsx ↔ OwnEnrollmentsList.
 */
function AppEnrollmentsPage() {
  const { pageSize } = Route.useSearch();
  const navigate = Route.useNavigate();

  return (
    <OwnEnrollmentsList
      pageSize={pageSize}
      onPageSizeChange={(n) =>
        navigate({ search: (prev) => ({ ...prev, pageSize: n }) })
      }
    />
  );
}

// Participant enrollments route — renders the student's own-enrollment list.
// The feature guard requires enrollment.view_own. See ROUTE_PERMISSIONS["/app/enrollments"].
export const Route = createFileRoute("/_authenticated/app/enrollments")({
  validateSearch: ownEnrollmentsSearchSchema,
  beforeLoad: ({ context }) => {
    requireRoutePermission(context.session, "/app/enrollments");
  },
  component: AppEnrollmentsPage,
});
