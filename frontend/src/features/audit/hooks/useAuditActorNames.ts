import { useQuery } from "@connectrpc/connect-query";
import { ProfileService } from "@/gen/profiles/v1/profiles_pb";

/**
 * Resolves display names for a set of actor user IDs found in audit logs.
 *
 * Mirrors `useDisplayNames` from section-enrollments: calls
 * `ProfileService.listDisplayNamesByIDs` and builds a userId → displayName map.
 * The query is skipped when actorIds is empty. When names are missing (e.g. the
 * actor no longer exists) the map has no entry for that ID — callers fall back
 * to the raw ID or a placeholder.
 *
 * @param actorIds - Deduplicated list of actor IDs present in the current view.
 * @returns A Map<userId, displayName> built from givenNames + lastNamePaternal.
 */
export function useAuditActorNames(actorIds: string[]): Map<string, string> {
  const query = useQuery(
    ProfileService.method.listDisplayNamesByIDs,
    { userIds: actorIds },
    { enabled: actorIds.length > 0 },
  );

  const nameMap = new Map<string, string>();
  if (query.data) {
    for (const dn of query.data.names) {
      nameMap.set(dn.userId, `${dn.givenNames} ${dn.lastNamePaternal}`);
    }
  }

  return nameMap;
}
