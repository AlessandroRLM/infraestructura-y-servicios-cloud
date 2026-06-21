import { Code, ConnectError } from "@connectrpc/connect";

/**
 * Maps an EnrollSection mutation error to a surface strategy.
 * Never surfaces raw error codes or service names.
 *
 * - FailedPrecondition → "precondition" (section full, window closed, unpaid, etc.)
 * - ResourceExhausted → "saturated" (admission controller throttling)
 * - AlreadyExists → "already_enrolled"
 * - Everything else → "transport" (caller shows a toast)
 */
export function mapEnrollSectionError(
  err: unknown,
): "precondition" | "saturated" | "already_enrolled" | "transport" {
  if (err instanceof ConnectError) {
    if (err.code === Code.FailedPrecondition) return "precondition";
    if (err.code === Code.ResourceExhausted) return "saturated";
    if (err.code === Code.AlreadyExists) return "already_enrolled";
  }
  return "transport";
}

/**
 * Maps a WithdrawSection mutation error to a surface strategy.
 * Never surfaces raw error codes or service names.
 *
 * - FailedPrecondition → "precondition" (not in_progress state)
 * - NotFound → "not_found"
 * - Everything else → "transport" (caller shows a toast)
 */
export function mapWithdrawSectionError(
  err: unknown,
): "precondition" | "not_found" | "transport" {
  if (err instanceof ConnectError) {
    if (err.code === Code.FailedPrecondition) return "precondition";
    if (err.code === Code.NotFound) return "not_found";
  }
  return "transport";
}
