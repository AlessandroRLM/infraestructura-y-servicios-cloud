import { Link, linkOptions } from "@tanstack/react-router";
import { RefreshCw } from "lucide-react";

/**
 * In-sidebar link that navigates to `/choose-area` so dual-eligible users
 * can switch between the admin and participant areas without logging out.
 *
 * Render this component only when the session is dual-eligible —
 * see `eligibleAreas(session).length === 2`.
 */
export function SwitchAreaControl() {
  return (
    <Link
      {...linkOptions({ to: "/choose-area" })}
      className="flex items-center gap-2 rounded-md px-3 py-2 text-muted-foreground text-sm transition-colors hover:bg-accent hover:text-foreground"
    >
      <RefreshCw className="size-4" aria-hidden />
      Cambiar área
    </Link>
  );
}
