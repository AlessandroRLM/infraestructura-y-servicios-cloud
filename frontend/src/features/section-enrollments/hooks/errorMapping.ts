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

/**
 * Maps an EnrollOwnSection mutation error to a human-readable Spanish message.
 * Never surfaces raw error codes or service names.
 *
 * - FailedPrecondition → window closed or capacity exceeded
 * - AlreadyExists → student is already enrolled in this section
 * - Everything else → generic transport/infra failure (toast)
 */
export function mapEnrollOwnSectionError(err: unknown): string {
  if (err instanceof ConnectError) {
    if (err.code === Code.FailedPrecondition)
      return "La ventana de inscripción está cerrada o la sección ya no tiene cupos disponibles.";
    if (err.code === Code.AlreadyExists)
      return "Ya estás inscrito en esta sección.";
  }
  return "No se pudo completar la inscripción. Intenta de nuevo.";
}
