import { createFileRoute } from "@tanstack/react-router";
import { requireRoutePermission } from "@/features/auth";
import {
  EnrollmentsPage,
  adminEnrollmentsSearchSchema,
} from "@/features/enrollments";

export const Route = createFileRoute("/_authenticated/admin/enrollments")({
  validateSearch: adminEnrollmentsSearchSchema,
  beforeLoad: ({ context }) => {
    requireRoutePermission(context.session, "/admin/enrollments");
  },
  component: EnrollmentsPage,
});
