// Foundational feature: other features and the app layer (router, routes,
// main) may import this public API; auth imports no other feature.

export {
  type Area,
  ADMIN_PERMISSIONS,
  eligibleAreas,
  isEligibleFor,
  PARTICIPANT_PERMISSIONS,
} from "./area";
export { readPreferredArea, writePreferredArea } from "./areaPreference";
export { bootstrapQueryOptions, SESSION_QUERY_KEY } from "./api/queries";
export { createRpcSessionSource } from "./api/rpc";
export { stubSessionSource } from "./api/stub";
export { LoginForm } from "./components/LoginForm";
export { LogoutButton } from "./components/LogoutButton";
export { SessionContext } from "./context/context";
export { SessionProvider } from "./context/provider";
export { requireAnyPermission, requireRoutePermission } from "./guards";
export { useLogin } from "./hooks/useLogin";
export { useLogout } from "./hooks/useLogout";
export { hasPermission, hasRole, useSession } from "./hooks/useSession";
export { PERMISSIONS, type Permission, ROLES, type Role } from "./permissions";
export {
  primaryRoleLabel,
  ROLE_LABELS,
  ROLE_PRIORITY,
  roleLabel,
  sortRolesByPriority,
} from "./roles";
export {
  type GuardedRoute,
  ROUTE_PERMISSIONS,
  routePermissions,
} from "./routePermissions";
export { resolveEntryTarget } from "./resolveEntryTarget";
export { loginSearchSchema } from "./schemas/search";
export type {
  AuthenticatedSession,
  SessionSource,
  SessionState,
} from "./types";
