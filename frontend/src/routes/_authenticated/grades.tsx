import { createFileRoute } from "@tanstack/react-router";
import { requireRoutePermission } from "@/features/auth";
import { GradesPage } from "@/features/grades";

export const Route = createFileRoute("/_authenticated/grades")({
  beforeLoad: ({ context }) => {
    requireRoutePermission(context.session, "/grades");
  },
  component: GradesPage,
});
