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
import { PageSizeSelector } from "@/core/components";
import { useOwnEnrollments } from "../hooks/useOwnEnrollments";
import { EnrollmentStatusBadge } from "./EnrollmentStatusBadge";

interface OwnEnrollmentsListProps {
  pageSize: number;
  onPageSizeChange: (n: number) => void;
}

/**
 * Presentational read-only list of the authenticated student's own enrollments.
 *
 * ADR-8: No filters, no actions column — students see all their enrollments
 * ordered as returned by the backend (year descending).
 *
 * Route-owned wrapper (app/enrollments.tsx) reads search params and passes
 * pageSize + onPageSizeChange as props, keeping this component free of any
 * direct Route import and avoiding circular dependencies.
 */
export function OwnEnrollmentsList({
  pageSize,
  onPageSizeChange,
}: OwnEnrollmentsListProps) {
  const {
    enrollments,
    isLoading,
    isError,
    refetch,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useOwnEnrollments(pageSize);

  return (
    <div className="space-y-6">
      <h1 className="font-semibold text-2xl tracking-tight">Mis matrículas</h1>

      <div className="flex flex-col gap-4">
        <div className="flex justify-end">
          <PageSizeSelector value={pageSize} onChange={onPageSizeChange} />
        </div>

        {isLoading && (
          <div
            role="status"
            className="space-y-2"
            aria-busy="true"
            aria-label="Cargando tus matrículas"
          >
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-12 w-full" />
            ))}
          </div>
        )}

        {!isLoading && isError && (
          <div
            className="rounded-md border border-destructive/50 p-4"
            role="alert"
          >
            <p className="text-destructive text-sm font-medium">
              No se pudieron cargar tus matrículas.
            </p>
            <Button
              variant="outline"
              size="sm"
              className="mt-3 gap-2"
              onClick={() => refetch()}
            >
              <RefreshCw className="size-4" aria-hidden />
              Reintentar
            </Button>
          </div>
        )}

        {!isLoading && !isError && enrollments.length === 0 && (
          <div className="flex flex-col items-center justify-center gap-4 rounded-md border border-dashed p-12 text-center">
            <p className="text-muted-foreground text-sm">
              Todavía no tienes matrículas.
            </p>
          </div>
        )}

        {!isLoading && !isError && enrollments.length > 0 && (
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Programa</TableHead>
                  <TableHead>Año</TableHead>
                  <TableHead>Estado</TableHead>
                  <TableHead>Creado</TableHead>
                  <TableHead>Pagado</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {enrollments.map((enrollment) => (
                  <TableRow key={enrollment.id}>
                    <TableCell className="font-medium">
                      {enrollment.programName}
                    </TableCell>
                    <TableCell>{enrollment.year}</TableCell>
                    <TableCell>
                      <EnrollmentStatusBadge status={enrollment.status} />
                    </TableCell>
                    <TableCell>
                      {new Date(enrollment.createdAt).toLocaleDateString(
                        "es-CL",
                      )}
                    </TableCell>
                    <TableCell>
                      {enrollment.paidAt
                        ? new Date(enrollment.paidAt).toLocaleDateString(
                            "es-CL",
                          )
                        : "—"}
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
                  await fetchNextPage({ throwOnError: true });
                } catch {
                  toast.error("No se pudieron cargar más matrículas.");
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
    </div>
  );
}
