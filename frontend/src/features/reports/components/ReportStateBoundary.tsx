import { Code, ConnectError } from "@connectrpc/connect";
import { AlertCircle, Loader2 } from "lucide-react";
import type { ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { TruncationBanner } from "./TruncationBanner";

/**
 * Discriminated truncation props — when truncated is true, truncatedTo is required.
 * This makes it impossible to render a truncated state without the row count.
 */
type TruncationProps =
  | { truncated: true; truncatedTo: number }
  | { truncated?: false; truncatedTo?: never };

type ReportStateBoundaryProps = TruncationProps & {
  /** True when no filter is set (query cannot be enabled). */
  filterSet: boolean;
  /** True when the backend query is in-flight. */
  isFetching: boolean;
  /** True when the PDF renderer is working. */
  isRendering: boolean;
  /** True when the query failed. */
  isError: boolean;
  /** The query error, if any. */
  error: Error | null;
  /** True when the query succeeded with zero rows. */
  isEmpty: boolean;
  /** Called when the user clicks "Reintentar". */
  onRetry: () => void;
  /** The content to render in the ready+truncated or ready state. */
  children: ReactNode;
};

/**
 * Standardized UX state boundary for all report tab panels.
 *
 * State priority order (mutually exclusive):
 *   1. No filter set            → neutral prompt
 *   2. Loading / generating     → spinner + text
 *   3. Error                    → human-readable message + Reintentar
 *   4. Empty                    → empty message (no iframe, no download)
 *   5. Ready + truncated        → TruncationBanner above children
 *   6. Ready                    → children
 *
 * Error messages are never raw gRPC codes or service names (per ux-rules).
 */
export function ReportStateBoundary({
  filterSet,
  isFetching,
  isRendering,
  isError,
  error,
  isEmpty,
  truncated,
  truncatedTo,
  onRetry,
  children,
}: ReportStateBoundaryProps) {
  // 1. No filter
  if (!filterSet) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center text-muted-foreground">
        <p className="text-sm">Selecciona un filtro para generar el reporte.</p>
      </div>
    );
  }

  // 2. Loading / generating
  if (isFetching || isRendering) {
    return (
      <div
        role="status"
        aria-busy="true"
        aria-label="Generando PDF"
        className="flex flex-col items-center justify-center gap-3 py-16"
      >
        <Loader2
          className="size-8 animate-spin text-muted-foreground"
          aria-hidden
        />
        <p className="text-sm text-muted-foreground">Generando PDF…</p>
      </div>
    );
  }

  // 3. Error
  if (isError) {
    const message = mapErrorMessage(error);
    return (
      <div
        role="alert"
        className="flex flex-col items-start gap-3 rounded-md border border-destructive/50 p-4"
      >
        <div className="flex items-center gap-2">
          <AlertCircle className="size-4 text-destructive" aria-hidden />
          <p className="text-sm font-medium text-destructive">{message}</p>
        </div>
        <Button variant="outline" size="sm" onClick={onRetry}>
          Reintentar
        </Button>
      </div>
    );
  }

  // 4. Empty
  if (isEmpty) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center text-muted-foreground">
        <p className="text-sm">No hay datos disponibles para este reporte.</p>
      </div>
    );
  }

  // 5 & 6. Ready (with optional truncation banner above content)
  return (
    <div className="flex flex-col gap-3">
      {truncated === true && <TruncationBanner count={truncatedTo} />}
      {children}
    </div>
  );
}

/**
 * Maps a query/render error to a safe, human-readable Spanish message.
 *
 * Uses structured ConnectError matching (error instanceof ConnectError + code enum)
 * instead of substring matching — never exposes raw error codes, gRPC status
 * integers, or service names.
 *
 * - Code.PermissionDenied → "no permission" copy (RF-6.2, RF-6.4)
 * - Code.Unauthenticated → generic copy (authentication is a separate concern;
 *   do NOT collapse into the permission message)
 * - Network/infra → connectivity copy
 * - Anything else → generic copy
 */
function mapErrorMessage(error: Error | null): string {
  if (error === null) {
    return "Ocurrió un error al generar el reporte. Inténtalo de nuevo.";
  }

  if (error instanceof ConnectError) {
    if (error.code === Code.PermissionDenied) {
      return "No tienes permiso para ver este reporte.";
    }
    // Code.Unauthenticated intentionally falls through to the generic message.
    // Network/transport codes:
    if (
      error.code === Code.Unavailable ||
      error.code === Code.DeadlineExceeded
    ) {
      return "No se pudo conectar al servidor. Verifica tu conexión e inténtalo de nuevo.";
    }
    return "Ocurrió un error al generar el reporte. Inténtalo de nuevo.";
  }

  // Non-ConnectError: network / browser fetch errors (retain existing coverage).
  const msg = error.message.toLowerCase();
  if (
    msg.includes("network") ||
    msg.includes("fetch") ||
    msg.includes("unavailable") ||
    msg.includes("timeout")
  ) {
    return "No se pudo conectar al servidor. Verifica tu conexión e inténtalo de nuevo.";
  }

  return "Ocurrió un error al generar el reporte. Inténtalo de nuevo.";
}
