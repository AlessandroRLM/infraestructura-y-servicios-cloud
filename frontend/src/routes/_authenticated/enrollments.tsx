import { createFileRoute } from "@tanstack/react-router";
import { requireAnyPermission } from "@/features/auth";
import { EnrollmentsPage } from "@/features/enrollments";

export const Route = createFileRoute("/_authenticated/enrollments")({
  beforeLoad: ({ context }) => {
    requireAnyPermission(context.session, ["enrollment.manage"]);
  },
  component: EnrollmentsPage,
});
