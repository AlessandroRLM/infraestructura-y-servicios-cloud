import type { SessionSource } from "./source";

export const stubSessionSource: SessionSource = {
  getSession: async () => ({
    user: null,
    roles: [],
    permissions: [],
    status: "unauthenticated",
  }),
};
