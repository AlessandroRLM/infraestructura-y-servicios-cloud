import { Loader2, RefreshCw } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { AuditLog } from "@/gen/audit_logs/v1/audit_logs_pb";

/** Formats an RFC3339 timestamp as dd/MM/yyyy HH:mm in the user's locale. */
function formatTimestamp(rfc3339: string): string {
  const d = new Date(rfc3339);
  if (Number.isNaN(d.getTime())) return rfc3339;
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${pad(d.getDate())}/${pad(d.getMonth() + 1)}/${d.getFullYear()} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

interface AuditLogsTableProps {
  logs: AuditLog[];
  isLoading: boolean;
  isError: boolean;
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  actorNames: Map<string, string>;
  onRefetch: () => void;
  onLoadMore: () => Promise<void>;
}

/**
 * Renders the paginated audit log feed as a table.
 *
 * Handles loading (skeleton), error+retry, empty, and "Cargar más" states —
 * mirroring the UsersTable pattern.
 */
export function AuditLogsTable({
  logs,
  isLoading,
  isError,
  hasNextPage,
  isFetchingNextPage,
  actorNames,
  onRefetch,
  onLoadMore,
}: AuditLogsTableProps) {
  return (
    <div className="flex flex-col gap-4">
      {isLoading && (
        <div
          role="status"
          aria-busy="true"
          aria-label="Cargando registros de auditoría"
          className="flex flex-col gap-2"
        >
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-10 w-full" />
          ))}
        </div>
      )}

      {!isLoading && isError && (
        <div
          className="rounded-md border border-destructive/50 p-4"
          role="alert"
        >
          <p className="text-destructive text-sm font-medium">
            No se pudo cargar el registro de auditoría.
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-3 gap-2"
            onClick={onRefetch}
          >
            <RefreshCw className="size-4" aria-hidden />
            Reintentar
          </Button>
        </div>
      )}

      {!isLoading && !isError && logs.length === 0 && (
        <div className="flex flex-col items-center justify-center gap-4 rounded-md border border-dashed p-12 text-center">
          <p className="text-muted-foreground text-sm">
            No se encontraron registros de auditoría.
          </p>
        </div>
      )}

      {!isLoading && !isError && logs.length > 0 && (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Fecha</TableHead>
                <TableHead>Actor</TableHead>
                <TableHead>Acción</TableHead>
                <TableHead>Entidad</TableHead>
                <TableHead>Detalle</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {logs.map((log) => (
                <TableRow key={log.id}>
                  <TableCell className="whitespace-nowrap">
                    {formatTimestamp(log.createdAt)}
                  </TableCell>
                  <TableCell>
                    {log.actorId
                      ? (actorNames.get(log.actorId) ?? log.actorId)
                      : "Sistema"}
                  </TableCell>
                  <TableCell>{log.action}</TableCell>
                  <TableCell>{log.entity}</TableCell>
                  <TableCell className="max-w-xs truncate">
                    {log.detail}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {hasNextPage && (
        <div className="flex justify-center">
          <Button
            variant="outline"
            onClick={async () => {
              try {
                await onLoadMore();
              } catch {
                toast.error("No se pudieron cargar más registros.");
              }
            }}
            disabled={isFetchingNextPage}
            className="gap-2"
          >
            {isFetchingNextPage && (
              <Loader2
                data-icon="inline-start"
                className="animate-spin"
                aria-hidden
              />
            )}
            Cargar más
          </Button>
        </div>
      )}
    </div>
  );
}
