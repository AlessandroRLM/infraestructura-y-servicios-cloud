import { useQuery } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { bootstrapQueryOptions } from "../api/queries";
import type {
  AuthenticatedSession,
  SessionSource,
  SessionState,
} from "../types";
import { SessionContext } from "./context";

interface SessionProviderProps {
  source: SessionSource;
  children: ReactNode;
}

const LOADING_SESSION: SessionState = { status: "loading" };
const UNAUTHENTICATED_SESSION: SessionState = { status: "unauthenticated" };
const ERROR_SESSION: SessionState = { status: "error" };

function deriveSession(
  isPending: boolean,
  isError: boolean,
  data: AuthenticatedSession | null | undefined,
): SessionState {
  if (isPending) {
    return LOADING_SESSION;
  }
  if (isError) {
    return ERROR_SESSION;
  }
  if (data) {
    return { status: "authenticated", ...data };
  }
  return UNAUTHENTICATED_SESSION;
}

export function SessionProvider({ source, children }: SessionProviderProps) {
  const { data, isPending, isError } = useQuery(bootstrapQueryOptions(source));
  const session = deriveSession(isPending, isError, data);

  return <SessionContext value={session}>{children}</SessionContext>;
}
