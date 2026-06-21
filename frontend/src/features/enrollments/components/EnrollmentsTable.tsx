import { Loader2, Plus, RefreshCw } from "lucide-react";
import { useState } from "react";
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
import { hasPermission, useSession } from "@/features/auth";
import type { Enrollment } from "@/gen/enrollment/v1/enrollment_pb";
import { Route } from "@/routes/_authenticated/admin/enrollments";
import { useEnrollments } from "../hooks/useEnrollments";
import { CancelEnrollmentDialog } from "./CancelEnrollmentDialog";
import { EnrollmentStatusBadge } from "./EnrollmentStatusBadge";
import { EnrollmentsFilterBar } from "./EnrollmentsFilterBar";
import { MarkPaidDialog } from "./MarkPaidDialog";

export function EnrollmentsTable() {
  const session = useSession();
  const canManage = hasPermission(session, "enrollment.manage");

  const { q, year, status, pageSize } = Route.useSearch();
  const navigate = Route.useNavigate();

  const {
    enrollments,
    isLoading,
    isError,
    refetch,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useEnrollments({ q, year, status, pageSize });

  const [markPaidEnrollment, setMarkPaidEnrollment] = useState<
    Enrollment | undefined
  >(undefined);
  const [cancelEnrollment, setCancelEnrollment] = useState<
    Enrollment | undefined
  >(undefined);

  return (
    <div className="flex flex-col gap-4">
      <EnrollmentsFilterBar
        q={q}
        year={year}
        status={status}
        pageSize={pageSize}
        onQChange={(v) => navigate({ search: (prev) => ({ ...prev, q: v }) })}
        onYearChange={(y) =>
          navigate({ search: (prev) => ({ ...prev, year: y }) })
        }
        onStatusChange={(s) =>
          navigate({ search: (prev) => ({ ...prev, status: s }) })
        }
        onPageSizeChange={(n) =>
          navigate({ search: (prev) => ({ ...prev, pageSize: n }) })
        }
      />

      {isLoading && (
        <div
          role="status"
          aria-busy="true"
          aria-label="Cargando matrículas"
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
            No se pudo cargar la lista de matrículas.
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
            Todavía no hay matrículas
          </p>
          {canManage && (
            <Button
              onClick={() =>
                navigate({
                  search: (prev) => ({ ...prev }),
                })
              }
              className="gap-2"
            >
              <Plus className="size-4" aria-hidden />
              Crear matrícula
            </Button>
          )}
        </div>
      )}

      {!isLoading && !isError && enrollments.length > 0 && (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Estudiante</TableHead>
                <TableHead>Programa</TableHead>
                <TableHead>Año</TableHead>
                <TableHead>Estado</TableHead>
                <TableHead>Creado</TableHead>
                <TableHead>Pagado</TableHead>
                {canManage && <TableHead className="w-[180px]" />}
              </TableRow>
            </TableHeader>
            <TableBody>
              {enrollments.map((enrollment) => (
                <EnrollmentRow
                  key={enrollment.id}
                  enrollment={enrollment}
                  canManage={canManage}
                  onMarkPaid={() => setMarkPaidEnrollment(enrollment)}
                  onCancel={() => setCancelEnrollment(enrollment)}
                />
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

      {markPaidEnrollment && (
        <MarkPaidDialog
          open
          onOpenChange={(open) => {
            if (!open) setMarkPaidEnrollment(undefined);
          }}
          enrollment={markPaidEnrollment}
        />
      )}

      {cancelEnrollment && (
        <CancelEnrollmentDialog
          open
          onOpenChange={(open) => {
            if (!open) setCancelEnrollment(undefined);
          }}
          enrollment={cancelEnrollment}
        />
      )}
    </div>
  );
}

// --- Row subcomponent ---

interface EnrollmentRowProps {
  enrollment: Enrollment;
  canManage: boolean;
  onMarkPaid: () => void;
  onCancel: () => void;
}

function EnrollmentRow({
  enrollment,
  canManage,
  onMarkPaid,
  onCancel,
}: EnrollmentRowProps) {
  const studentDisplay =
    enrollment.studentName.length > 0
      ? enrollment.studentName
      : enrollment.studentId.slice(0, 8);

  const programDisplay =
    enrollment.programName.length > 0
      ? enrollment.programName
      : enrollment.programId.slice(0, 8);

  const paidAtDisplay = enrollment.paidAt
    ? new Date(enrollment.paidAt).toLocaleDateString("es-CL")
    : "—";

  const createdAtDisplay = enrollment.createdAt
    ? new Date(enrollment.createdAt).toLocaleDateString("es-CL")
    : "—";

  return (
    <TableRow>
      <TableCell>{studentDisplay}</TableCell>
      <TableCell>{programDisplay}</TableCell>
      <TableCell>{enrollment.year}</TableCell>
      <TableCell>
        <EnrollmentStatusBadge status={enrollment.status} />
      </TableCell>
      <TableCell className="text-muted-foreground text-sm">
        {createdAtDisplay}
      </TableCell>
      <TableCell className="text-muted-foreground text-sm">
        {paidAtDisplay}
      </TableCell>
      {canManage && (
        <TableCell className="flex gap-2 justify-end">
          {enrollment.status === "pending" && (
            <Button variant="ghost" size="sm" onClick={onMarkPaid}>
              Marcar pagada
            </Button>
          )}
          {(enrollment.status === "pending" ||
            enrollment.status === "paid") && (
            <Button
              variant="ghost"
              size="sm"
              className="text-destructive hover:text-destructive"
              onClick={onCancel}
            >
              Cancelar
            </Button>
          )}
        </TableCell>
      )}
    </TableRow>
  );
}
