import { createFileRoute } from "@tanstack/react-router";
import { AccessControlPage } from "@/features/access-control";
import { requireRoutePermission } from "@/features/auth";

export const Route = createFileRoute("/_authenticated/access-control")({
  beforeLoad: ({ context }) => {
    requireRoutePermission(context.session, "/access-control");
  },
  component: AccessControlPage,
});
