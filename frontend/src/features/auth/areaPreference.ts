import type { Area } from "./area";

const STORAGE_KEY = "iyc.preferredArea";

const VALID_AREAS: ReadonlySet<string> = new Set<Area>(["admin", "participant"]);

/**
 * Reads the stored area preference from localStorage.
 * Returns `null` when the key is absent, the stored value is not a valid
 * `Area`, or localStorage throws (private-browsing / storage disabled).
 */
export function readPreferredArea(): Area | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw !== null && VALID_AREAS.has(raw)) {
      return raw as Area;
    }
    return null;
  } catch {
    return null;
  }
}

/**
 * Writes `area` to localStorage under the preferred-area key.
 * Silently swallows storage errors (private-browsing / quota exceeded).
 */
export function writePreferredArea(area: Area): void {
  try {
    localStorage.setItem(STORAGE_KEY, area);
  } catch {
    // Storage unavailable — the caller continues without crashing.
  }
}
