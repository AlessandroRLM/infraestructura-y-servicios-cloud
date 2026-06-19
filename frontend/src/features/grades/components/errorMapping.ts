import { Code, ConnectError } from "@connectrpc/connect";

/**
 * Discriminated return type for scheme mutation errors.
 * The caller renders the appropriate user-facing message per kind:
 * - "precondition": the scheme is locked because grades exist; show inline banner.
 * - "already-exists": defensive — create was called on a course that already has a scheme.
 * - "generic": transport or unexpected error; caller shows a toast.
 */
export type SchemeErrorKind = "precondition" | "already-exists" | "generic";

/**
 * Discriminated return type for grade write errors (RecordGrade / OverrideGrade).
 * - "conflict": CodeAborted — stale expected_version; triggers row-scoped refetch + merge.
 * - "generic": any other error; caller shows per-cell inline error.
 */
export type GradeWriteErrorKind = "conflict" | "generic";

/**
 * Maps a RecordGrade or OverrideGrade error to a GradeWriteErrorKind.
 * Never surfaces raw gRPC codes, stack traces, or service names to the UI.
 *
 * - CodeAborted (stale expected_version) → "conflict"
 *   Caller triggers row-scoped refetch and surfaces:
 *   "Otro usuario modificó esta nota. Recargá para ver el valor actualizado."
 * - Anything else → "generic"
 *   Caller shows per-cell inline error.
 *
 * @param err - The error thrown by a useMutation call, or any unknown value.
 */
export function mapGradeWriteError(err: unknown): GradeWriteErrorKind {
  if (err instanceof ConnectError && err.code === Code.Aborted) {
    return "conflict";
  }
  return "generic";
}

/**
 * Maps a scheme mutation error to a SchemeErrorKind.
 * Never surfaces raw gRPC codes, stack traces, or service names to the UI.
 *
 * - FailedPrecondition (RecreateEvaluationScheme with grades) → "precondition"
 *   Caller renders: "Este curso ya tiene notas registradas. No es posible reemplazar el esquema."
 * - AlreadyExists (CreateEvaluationScheme race condition) → "already-exists"
 *   Caller renders: "El esquema ya existe. Recarga la página e intenta de nuevo."
 * - Anything else → "generic"
 *   Caller renders a toast: "No se pudo guardar el esquema. Inténtalo de nuevo."
 *
 * @param err - The error thrown by a useMutation call, or any unknown value.
 */
export function mapSchemeError(err: unknown): SchemeErrorKind {
  if (err instanceof ConnectError) {
    if (err.code === Code.FailedPrecondition) return "precondition";
    if (err.code === Code.AlreadyExists) return "already-exists";
  }
  return "generic";
}
