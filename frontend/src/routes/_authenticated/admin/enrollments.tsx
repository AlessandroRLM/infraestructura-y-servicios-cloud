import { createFileRoute } from "@tanstack/react-router";
import { requireRoutePermission } from "@/features/auth";
import { EnrollmentsPage } from "@/features/enrollments";
import { adminEnrollmentsSearchSchema } from "@/features/enrollments/schemas/search";

export const Route = createFileRoute("/_authenticated/admin/enrollments")({
  validateSearch: adminEnrollmentsSearchSchema,
  beforeLoad: ({ context }) => {
    requireRoutePermission(context.session, "/admin/enrollments");
  },
  component: EnrollmentsPage,
});
