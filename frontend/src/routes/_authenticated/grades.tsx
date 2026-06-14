import { createFileRoute } from "@tanstack/react-router";
import { requireAnyPermission } from "@/features/auth";
import { GradesPage } from "@/features/grades";

export const Route = createFileRoute("/_authenticated/grades")({
  beforeLoad: ({ context }) => {
    requireAnyPermission(context.session, [
      "grades.read",
      "grades.write",
      "grades.view_own",
    ]);
  },
  component: GradesPage,
});
