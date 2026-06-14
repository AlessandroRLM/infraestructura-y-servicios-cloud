import { Code, ConnectError } from "@connectrpc/connect";
import type { UseFormSetError } from "react-hook-form";
import type { CourseFormValues } from "../schemas/course";
import type { ProgramFormValues } from "../schemas/program";

/**
 * Maps a mutation error to either an inline field error or a toast signal.
 * Never surfaces raw error text or codes to the UI.
 */
export function mapProgramMutationError(
  err: unknown,
  setError: UseFormSetError<ProgramFormValues>,
): "handled-inline" | "toast" {
  if (err instanceof ConnectError) {
    if (err.code === Code.AlreadyExists || err.code === Code.InvalidArgument) {
      setError("code", { message: "Ya existe una carrera con ese código" });
      return "handled-inline";
    }
  }
  return "toast";
}

/**
 * Maps a mutation error to either an inline field error or a toast signal.
 * Never surfaces raw error text or codes to the UI.
 */
export function mapCourseMutationError(
  err: unknown,
  setError: UseFormSetError<CourseFormValues>,
): "handled-inline" | "toast" {
  if (err instanceof ConnectError) {
    if (err.code === Code.AlreadyExists || err.code === Code.InvalidArgument) {
      setError("code", { message: "Ya existe una asignatura con ese código" });
      return "handled-inline";
    }
  }
  return "toast";
}

/**
 * Maps a delete mutation error to its surface strategy.
 * FailedPrecondition = entity has dependents, shown inline in the dialog.
 */
export function mapDeleteError(err: unknown): "precondition" | "transport" {
  if (err instanceof ConnectError && err.code === Code.FailedPrecondition) {
    return "precondition";
  }
  return "transport";
}
