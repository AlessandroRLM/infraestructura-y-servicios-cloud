import type { Area } from "./area";

/** Destinations the entry redirect can send the user to. */
export type EntryTarget =
  | "/admin"
  | "/app"
  | "/choose-area"
  | "/forbidden";

const AREA_ROOTS: Record<Area, EntryTarget> = {
  admin: "/admin",
  participant: "/app",
};

/**
 * Resolves the correct entry destination given the session's eligible areas
 * and the stored area preference.
 *
 * - 0 eligible areas → `/forbidden`
 * - 1 eligible area → that area's root (preference is irrelevant)
 * - 2 eligible areas + valid stored preference → that area's root
 * - 2 eligible areas + absent/invalid preference → `/choose-area`
 *
 * Pure function — no side effects.
 */
export function resolveEntryTarget(
  areas: Area[],
  preferred: Area | null,
): EntryTarget {
  if (areas.length === 0) return "/forbidden";
  if (areas.length === 1) return AREA_ROOTS[areas[0]];

  // Dual-eligible: honour preference only when it refers to an area the user
  // is actually eligible for.
  if (preferred !== null && (areas as string[]).includes(preferred)) {
    return AREA_ROOTS[preferred];
  }

  return "/choose-area";
}
