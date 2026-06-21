import { createFileRoute, redirect } from "@tanstack/react-router";
import {
  firstAccessibleNavTarget,
  PARTICIPANT_NAV,
} from "@/components/layout/AppSidebar";

// Redirects to the first participant nav entry accessible to the current session.
// Using the session resolved by the parent _authenticated/route.tsx beforeLoad
// so no second fetch is needed.
export const Route = createFileRoute("/_authenticated/app/")({
  beforeLoad: ({ context }) => {
    const session = context.session;
    const target = firstAccessibleNavTarget(
      { ...session, status: "authenticated" },
      PARTICIPANT_NAV,
    );
    if (target) {
      throw redirect(target);
    }
    throw redirect({ to: "/forbidden" });
  },
});
