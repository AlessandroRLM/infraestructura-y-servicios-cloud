export interface AuthenticatedSession {
  userId: string;
  email: string;
  roles: string[];
  permissions: string[];
}

// `loading` distinguishes "not yet known" from "known to be logged out" so
// guards can wait for the bootstrap instead of redirecting prematurely.
// `error` distinguishes infra failure from a legitimate logged-out state so
// the UI can surface a recoverable error instead of silently landing on /login.
export type SessionState =
  | { status: "loading" }
  | { status: "unauthenticated" }
  | { status: "error" }
  | ({ status: "authenticated" } & AuthenticatedSession);

export interface SessionSource {
  /** Resolves the current session; `null` means no active session. */
  getSession(): Promise<AuthenticatedSession | null>;
}
