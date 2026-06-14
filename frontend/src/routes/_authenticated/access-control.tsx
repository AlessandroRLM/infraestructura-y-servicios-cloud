import { createFileRoute } from "@tanstack/react-router";
import { AccessControlPage } from "@/features/access-control";
import { requireAnyPermission } from "@/features/auth";

export const Route = createFileRoute("/_authenticated/access-control")({
  beforeLoad: ({ context }) => {
    requireAnyPermission(context.session, ["users.manage"]);
  },
  component: AccessControlPage,
});
