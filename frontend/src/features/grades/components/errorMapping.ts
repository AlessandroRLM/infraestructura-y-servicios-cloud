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
