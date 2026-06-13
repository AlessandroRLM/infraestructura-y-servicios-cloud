// Foundational feature: other features and the app layer (router, routes,
// main) may import this public API; auth imports no other feature.

export { bootstrapQueryOptions, SESSION_QUERY_KEY } from "./api/queries";
export { createRpcSessionSource } from "./api/rpc";
export { stubSessionSource } from "./api/stub";
export { LoginForm } from "./components/LoginForm";
export { LogoutButton } from "./components/LogoutButton";
export { SessionContext } from "./context/context";
export { SessionProvider } from "./context/provider";
export { useLogin } from "./hooks/useLogin";
export { useLogout } from "./hooks/useLogout";
export { hasPermission, hasRole, useSession } from "./hooks/useSession";
export { PERMISSIONS, type Permission, ROLES, type Role } from "./permissions";
export { loginSearchSchema } from "./schemas/search";
export type {
  AuthenticatedSession,
  SessionSource,
  SessionState,
} from "./types";
