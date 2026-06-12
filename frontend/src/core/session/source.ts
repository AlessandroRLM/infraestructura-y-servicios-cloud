export interface SessionData {
  user: { id: string; email: string } | null;
  roles: string[];
  permissions: string[];
  status: "authenticated" | "unauthenticated" | "loading";
}

export interface SessionSource {
  getSession(): Promise<SessionData>;
}
