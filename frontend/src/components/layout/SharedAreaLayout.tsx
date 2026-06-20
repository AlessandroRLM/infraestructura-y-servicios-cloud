import { Outlet } from "@tanstack/react-router";
import type { ReactNode } from "react";
import { eligibleAreas, readPreferredArea, useSession } from "@/features/auth";
import type { Area } from "@/features/auth";
import { ADMIN_NAV, AppSidebar, PARTICIPANT_NAV } from "./AppSidebar";

/**
 * Resolves the active area for a shared route (e.g. /profile).
 *
 * Priority:
 *   1. readPreferredArea() — if the stored value is in eligibleAreas()
 *   2. First entry in eligibleAreas() — fallback
 *   3. "participant" — last resort when session has zero eligible areas
 */
function resolveActiveArea(areas: Area[], preferred: Area | null): Area {
  if (preferred !== null && areas.includes(preferred)) return preferred;
  return areas[0] ?? "participant";
}

interface SharedAreaLayoutProps {
  /**
   * When provided the children are rendered instead of <Outlet />.
   * Use this for leaf routes that don't have nested children.
   */
  children?: ReactNode;
}

/**
 * Layout shell for shared authenticated routes (e.g. /profile).
 *
 * Picks the correct area sidebar based on the user's preferred area
 * (or the first eligible area as a fallback), matching the pattern
 * used by AdminLayout and ParticipantLayout.
 *
 * When used as a route layout (with nested children via file-based routing),
 * it renders <Outlet />. When used as a direct wrapper, pass children.
 */
export function SharedAreaLayout({ children }: SharedAreaLayoutProps) {
  const session = useSession();
  const isAuth = session.status === "authenticated";

  const areas = isAuth ? eligibleAreas(session) : [];
  const preferred = readPreferredArea();
  const activeArea = resolveActiveArea(areas, preferred);

  const activeNav = activeArea === "admin" ? ADMIN_NAV : PARTICIPANT_NAV;
  const isDualEligible = areas.length === 2;

  return (
    <div className="flex h-svh">
      <AppSidebar nav={activeNav} showSwitchArea={isDualEligible} />
      <main className="flex-1 overflow-y-auto p-8">
        {children ?? <Outlet />}
      </main>
    </div>
  );
}
