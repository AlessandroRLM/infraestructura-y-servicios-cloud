// Foundational feature: other features and the app layer (router, routes,
// main) may import this public API; auth imports no other feature.

export { bootstrapQueryOptions, SESSION_QUERY_KEY } from "./api/queries";
export { createRpcSessionSource } from "./api/rpc";
export { stubSessionSource } from "./api/stub";
export { AuthPage } from "./components/AuthPage";
export { SessionContext } from "./context/context";
export { SessionProvider } from "./context/provider";
export { hasPermission, hasRole, useSession } from "./hooks/useSession";
export { PERMISSIONS, type Permission, ROLES, type Role } from "./permissions";
export type {
  AuthenticatedSession,
  SessionSource,
  SessionState,
} from "./types";
