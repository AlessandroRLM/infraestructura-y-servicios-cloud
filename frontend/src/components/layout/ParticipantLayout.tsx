import { Outlet } from "@tanstack/react-router";
import { eligibleAreas, useSession } from "@/features/auth";
import { AppSidebar, PARTICIPANT_NAV } from "./AppSidebar";

/** Layout for the /app area. Renders the participant sidebar and the page outlet. */
export function ParticipantLayout() {
  const session = useSession();
  // Show the switch-area control only for dual-eligible users (admin + participant).
  const isDualEligible =
    session.status === "authenticated" && eligibleAreas(session).length === 2;

  return (
    <div className="flex h-svh">
      <AppSidebar nav={PARTICIPANT_NAV} showSwitchArea={isDualEligible} />
      <main className="flex-1 overflow-y-auto p-8">
        <Outlet />
      </main>
    </div>
  );
}
