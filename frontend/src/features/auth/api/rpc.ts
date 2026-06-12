import {
  Code,
  ConnectError,
  createClient,
  type Transport,
} from "@connectrpc/connect";
import { AuthService } from "@/gen/auth/v1/auth_pb";
import type { SessionSource } from "../types";

export function createRpcSessionSource(transport: Transport): SessionSource {
  const client = createClient(AuthService, transport);
  return {
    async getSession() {
      try {
        const session = await client.getSession({});
        return {
          userId: session.userId,
          email: session.email,
          roles: session.roles,
          permissions: session.permissions,
        };
      } catch (error) {
        // The backend signals "no active session" (missing/expired/invalid
        // cookie) with CodeUnauthenticated; any other error is infra failure.
        if (
          error instanceof ConnectError &&
          error.code === Code.Unauthenticated
        ) {
          return null;
        }
        throw error;
      }
    },
  };
}
