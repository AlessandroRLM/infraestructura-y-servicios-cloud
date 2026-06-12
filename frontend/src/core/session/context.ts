import { createContext, useContext } from "react";
import type { SessionData } from "./source";

export type SessionContext = SessionData;

export const SessionCtx = createContext<SessionContext | null>(null);

export function useSessionContext(): SessionContext {
  const ctx = useContext(SessionCtx);
  if (!ctx) {
    throw new Error("useSessionContext must be used within a SessionProvider");
  }
  return ctx;
}
