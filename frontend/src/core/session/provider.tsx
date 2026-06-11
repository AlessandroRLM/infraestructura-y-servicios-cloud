import { useQuery } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { SessionCtx } from "./context";
import { bootstrapQueryOptions } from "./queries";
import type { SessionData, SessionSource } from "./source";

/**
 * Pending / unauthenticated fallback used before the bootstrap query resolves.
 * This is type-honest: SessionData already models the unauthenticated state.
 * Guards and gating helpers handle this value correctly (hasPermission/hasRole
 * return false; the authenticated route guard redirects to /login).
 */
const UNAUTHENTICATED_SESSION: SessionData = {
  user: null,
  roles: [],
  permissions: [],
  status: "unauthenticated",
};

interface SessionProviderProps {
  /** The source used to resolve the session (RPC or stub for testing). */
  source: SessionSource;
  children: ReactNode;
}

/**
 * Provides session state to the component tree via SessionCtx.
 *
 * TanStack Query is the single source of truth. This component derives the
 * session by subscribing to the bootstrap query (no useState, no useEffect,
 * no internal state). Duplicate useQuery calls with the same query key
 * (staleTime: Infinity) deduplicate to the same cache entry — the router
 * context can safely call useQuery with the same key without a second fetch.
 *
 * Usage:
 *   <SessionProvider source={source}>
 *     <App />
 *   </SessionProvider>
 *
 * Note: login/logout mutations that need to invalidate the session query live
 * in features/auth (Connect mutations that call queryClient.invalidateQueries
 * with SESSION_QUERY_KEY). They are NOT part of this provider.
 */
export function SessionProvider({ source, children }: SessionProviderProps) {
  const { data } = useQuery(bootstrapQueryOptions(source));
  const session = data ?? UNAUTHENTICATED_SESSION;

  return <SessionCtx.Provider value={session}>{children}</SessionCtx.Provider>;
}
