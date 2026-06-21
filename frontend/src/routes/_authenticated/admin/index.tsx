import { createFileRoute, redirect } from "@tanstack/react-router";
import {
  ADMIN_NAV,
  firstAccessibleNavTarget,
} from "@/components/layout/AppSidebar";

// Redirects to the first admin nav entry accessible to the current session.
// Using the session resolved by the parent _authenticated/route.tsx beforeLoad
// so no second fetch is needed.
export const Route = createFileRoute("/_authenticated/admin/")({
  beforeLoad: ({ context }) => {
    const session = context.session;
    const target = firstAccessibleNavTarget(
      { ...session, status: "authenticated" },
      ADMIN_NAV,
    );
    if (target) {
      throw redirect(target);
    }
    throw redirect({ to: "/forbidden" });
  },
});
