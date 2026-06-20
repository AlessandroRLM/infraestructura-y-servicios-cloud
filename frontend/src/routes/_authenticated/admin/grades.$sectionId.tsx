import {
  createFileRoute,
  useLocation,
  useNavigate,
} from "@tanstack/react-router";
import { requireRoutePermission } from "@/features/auth";
import { GradesSectionPage } from "@/features/grades/components/GradesSectionPage";

// Admin grades section sub-route — renders the grade recording grid for a specific section.
// The section param is deep-linkable and refresh-stable: on click-through the full
// TeachingSection is passed via router navigation state; on hard refresh it is resolved
// from the ListOwnSections hook (same data source as the parent table).
// See ROUTE_PERMISSIONS["/admin/grades/$sectionId"].
export const Route = createFileRoute("/_authenticated/admin/grades/$sectionId")(
  {
    beforeLoad: ({ context }) => {
      requireRoutePermission(context.session, "/admin/grades/$sectionId");
    },
    component: RouteComponent,
  },
);

function RouteComponent() {
  const { sectionId } = Route.useParams();
  const location = useLocation();
  const navigate = useNavigate();

  return (
    <GradesSectionPage
      sectionId={sectionId}
      locationState={location.state}
      onBack={() => navigate({ to: "/admin/grades" })}
    />
  );
}
