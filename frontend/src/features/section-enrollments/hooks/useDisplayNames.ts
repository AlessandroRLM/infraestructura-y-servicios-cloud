import { useQuery } from "@connectrpc/connect-query";
import { ProfileService } from "@/gen/profiles/v1/profiles_pb";

/**
 * Resolves display names for a set of student user IDs.
 *
 * Mirrors the grades/useSectionGrid pattern: calls
 * ProfileService.listDisplayNamesByIDs and builds a userId → displayName map.
 * The query is skipped when userIds is empty. When the profile.view_names
 * permission gate is not met (or names are missing), the map simply has no
 * entry for that ID — callers fall back to the raw ID or its slice.
 *
 * @param userIds - Deduplicated list of user IDs present in the current view.
 * @returns A Map<userId, displayName> built from givenNames + lastNamePaternal.
 */
export function useDisplayNames(userIds: string[]): Map<string, string> {
  const displayNamesQuery = useQuery(
    ProfileService.method.listDisplayNamesByIDs,
    { userIds },
    { enabled: userIds.length > 0 },
  );

  const nameMap = new Map<string, string>();
  if (displayNamesQuery.data) {
    for (const dn of displayNamesQuery.data.names) {
      nameMap.set(dn.userId, `${dn.givenNames} ${dn.lastNamePaternal}`);
    }
  }

  return nameMap;
}
