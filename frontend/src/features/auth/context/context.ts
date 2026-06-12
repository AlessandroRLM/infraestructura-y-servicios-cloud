import { createContext } from "react";
import type { SessionState } from "../types";

export const SessionContext = createContext<SessionState | null>(null);
