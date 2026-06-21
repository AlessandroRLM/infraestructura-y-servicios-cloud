import { createFileRoute } from "@tanstack/react-router";
import { requireRoutePermission } from "@/features/auth";
import { SectionEnrollmentsPage } from "@/features/section-enrollments";
import { sectionEnrollmentsSearchSchema } from "@/features/section-enrollments/schemas/search";

function SectionEnrollmentsRoute() {
  const { q, pageSize } = Route.useSearch();
  const navigate = Route.useNavigate();

  return (
    <SectionEnrollmentsPage
      q={q}
      pageSize={pageSize}
      onQueryChange={(v) => navigate({ search: (prev) => ({ ...prev, q: v }) })}
      onPageSizeChange={(n) =>
        navigate({ search: (prev) => ({ ...prev, pageSize: n }) })
      }
    />
  );
}

export const Route = createFileRoute(
  "/_authenticated/admin/section-enrollments",
)({
  validateSearch: sectionEnrollmentsSearchSchema,
  beforeLoad: ({ context }) => {
    requireRoutePermission(context.session, "/admin/section-enrollments");
  },
  component: SectionEnrollmentsRoute,
});
