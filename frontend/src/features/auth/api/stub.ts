import type { SessionSource } from "../types";

export const stubSessionSource: SessionSource = {
  getSession: async () => null,
};
