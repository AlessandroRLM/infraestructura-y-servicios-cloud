import { createFileRoute, redirect } from "@tanstack/react-router";
import { ParticipantLayout } from "@/components/layout/ParticipantLayout";

// Area-eligibility guard for every /app/* route. Runs after the parent
// _authenticated beforeLoad has put { session, eligibility } on the context.
//
// Ineligible users are sent to the admin area when they have access
// there, or to /forbidden when they have no area at all.
export const Route = createFileRoute("/_authenticated/app")({
  beforeLoad: ({ context }) => {
    const { eligibility } = context;
    if (!eligibility.includes("participant")) {
      if (eligibility.includes("admin")) {
        throw redirect({ to: "/admin" });
      }
      throw redirect({ to: "/forbidden" });
    }
  },
  component: ParticipantLayout,
});
