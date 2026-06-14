import { Code, ConnectError } from "@connectrpc/connect";

/**
 * Maps a revoke error to its surface strategy.
 * FailedPrecondition = last-admin guard, shown inline in AlertDialog.
 */
export function mapRevokeError(err: unknown): "last-admin" | "transport" {
  if (err instanceof ConnectError && err.code === Code.FailedPrecondition) {
    return "last-admin";
  }
  return "transport";
}
