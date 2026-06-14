import { Code, ConnectError } from "@connectrpc/connect";
import type { UseFormSetError } from "react-hook-form";
import type { ProfileFormValues } from "../schemas/profile";

/**
 * Routes mutation errors to an inline field error or a toast signal.
 * Never surfaces raw error codes or backend internals to the user.
 */
export function mapProfileMutationError(
  err: unknown,
  setError: UseFormSetError<ProfileFormValues>,
): "handled-inline" | "toast" {
  if (err instanceof ConnectError && err.code === Code.InvalidArgument) {
    setError("birthDate", { message: "Fecha inválida (formato AAAA-MM-DD)" });
    return "handled-inline";
  }
  return "toast";
}
