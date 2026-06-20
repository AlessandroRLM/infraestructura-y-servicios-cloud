import { Code, ConnectError } from "@connectrpc/connect";
import type { UseFormSetError, FieldValues } from "react-hook-form";

/**
 * Maps a CreateEnrollment mutation error to either an inline field error or a
 * toast signal. Never surfaces raw error text, codes, or service names to the UI.
 *
 * AlreadyExists / InvalidArgument → inline domain message set on "root".
 * Everything else → "toast" signal (caller shows a toast).
 */
export function mapCreateEnrollmentError(
  err: unknown,
  setError?: UseFormSetError<FieldValues>,
): "handled-inline" | "toast" {
  if (err instanceof ConnectError) {
    if (err.code === Code.AlreadyExists || err.code === Code.InvalidArgument) {
      setError?.("root", {
        message:
          "Ya existe una matrícula para este estudiante, programa y año.",
      });
      return "handled-inline";
    }
  }
  return "toast";
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
