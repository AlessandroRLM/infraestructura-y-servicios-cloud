import { createFileRoute, redirect } from "@tanstack/react-router";
import { AdminLayout } from "@/components/layout/AdminLayout";

// Area-eligibility guard for every /admin/* route. Runs after the parent
// _authenticated beforeLoad has put { session, eligibility } on the context.
//
// Ineligible users are sent to the participant area when they have access
// there, or to /forbidden when they have no area at all.
export const Route = createFileRoute("/_authenticated/admin")({
  beforeLoad: ({ context }) => {
    const { eligibility } = context;
    if (!eligibility.includes("admin")) {
      if (eligibility.includes("participant")) {
        throw redirect({ to: "/app" });
      }
      throw redirect({ to: "/forbidden" });
    }
  },
  component: AdminLayout,
});
