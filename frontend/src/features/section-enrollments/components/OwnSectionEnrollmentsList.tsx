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
import { useOwnSectionEnrollments } from "../hooks/useOwnSectionEnrollments";
import { SectionEnrollmentStatusBadge } from "./SectionEnrollmentStatusBadge";

interface OwnSectionEnrollmentsListProps {
  pageSize: number;
  onPageSizeChange: (n: number) => void;
}

/**
 * Presentational list of the authenticated student's own section enrollments.
 * Read-only — shows section_id, status, registered_at, and final_grade.
 *
 * Route-owned wrapper (app/section-enrollments.tsx) reads search params and passes
 * pageSize + onPageSizeChange as props, keeping this component free of any
 * direct Route import and avoiding circular dependencies.
 */
export function OwnSectionEnrollmentsList({
  pageSize,
  onPageSizeChange,
}: OwnSectionEnrollmentsListProps) {
  const {
    sectionEnrollments,
    isLoading,
    isError,
    refetch,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useOwnSectionEnrollments(pageSize);

  return (
    <div className="space-y-6">
      <h1 className="font-semibold text-2xl tracking-tight">Mis secciones</h1>

      <div className="flex flex-col gap-4">
        <div className="flex justify-end">
          <PageSizeSelector value={pageSize} onChange={onPageSizeChange} />
        </div>

        {isLoading && (
          <div
            role="status"
            className="space-y-2"
            aria-busy="true"
            aria-label="Cargando tus inscripciones"
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
              No se pudieron cargar tus inscripciones.
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

        {!isLoading && !isError && sectionEnrollments.length === 0 && (
          <div className="flex flex-col items-center justify-center gap-4 rounded-md border border-dashed p-12 text-center">
            <p className="text-muted-foreground text-sm">
              Todavía no tienes inscripciones a secciones.
            </p>
          </div>
        )}

        {!isLoading && !isError && sectionEnrollments.length > 0 && (
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Sección</TableHead>
                  <TableHead>Estado</TableHead>
                  <TableHead>Inscrito</TableHead>
                  <TableHead>Nota final</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {sectionEnrollments.map((se) => (
                  <TableRow key={se.id}>
                    <TableCell className="font-medium">
                      {se.courseName ? (
                        <div className="flex flex-col">
                          <span>
                            {se.courseName}
                            {se.courseCode ? ` (${se.courseCode})` : ""}
                          </span>
                          <span className="text-muted-foreground text-xs">
                            {se.periodYear} · Semestre {se.periodTerm}
                          </span>
                        </div>
                      ) : (
                        se.sectionId
                      )}
                    </TableCell>
                    <TableCell>
                      <SectionEnrollmentStatusBadge status={se.status} />
                    </TableCell>
                    <TableCell>
                      {se.registeredAt
                        ? new Date(se.registeredAt).toLocaleDateString("es-CL")
                        : "—"}
                    </TableCell>
                    <TableCell>{se.finalGrade || "—"}</TableCell>
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
                  toast.error("No se pudieron cargar más inscripciones.");
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
