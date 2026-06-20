import { Code, ConnectError } from "@connectrpc/connect";

/**
 * Maps a CreateEnrollment mutation error to a Spanish inline message to show in
 * the dialog, or null when the caller should fall back to a generic toast.
 * Never surfaces raw error text, codes, or service names to the UI.
 *
 * - AlreadyExists / InvalidArgument → duplicate-enrollment message.
 * - FailedPrecondition → quota message ("cupo lleno" vs "sin cupo definido").
 *   On the create path, FailedPrecondition is always a quota precondition.
 * - Everything else → null (transport; caller shows a toast).
 */
export function mapCreateEnrollmentError(err: unknown): string | null {
  if (err instanceof ConnectError) {
    if (err.code === Code.AlreadyExists || err.code === Code.InvalidArgument) {
      return "Ya existe una matrícula para este estudiante, programa y año.";
    }
    if (err.code === Code.FailedPrecondition) {
      // The backend distinguishes a full quota from a missing one; both arrive
      // as FailedPrecondition, so disambiguate on the (stable) domain message.
      return /full/i.test(err.rawMessage)
        ? "El cupo de matrícula para este programa y año está completo."
        : "No hay cupo de matrícula definido para este programa y año.";
    }
  }
  return null;
}

/**
 * Maps a MarkEnrollmentPaid or CancelEnrollment mutation error to a surface
 * strategy. Never surfaces raw error codes.
 *
 * FailedPrecondition → "precondition" (caller shows inline message in dialog).
 * Everything else → "transport" (caller shows a toast).
 */
export function mapLifecycleError(err: unknown): "precondition" | "transport" {
  if (err instanceof ConnectError && err.code === Code.FailedPrecondition) {
    return "precondition";
  }
  return "transport";
}
