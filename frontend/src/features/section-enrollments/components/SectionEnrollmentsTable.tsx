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
import type { SectionEnrollment } from "@/gen/section_enrollment/v1/section_enrollment_pb";
import { useDisplayNames } from "../hooks/useDisplayNames";
import { useSectionEnrollments } from "../hooks/useSectionEnrollments";
import { EnrollSectionDialog } from "./EnrollSectionDialog";
import { SectionEnrollmentStatusBadge } from "./SectionEnrollmentStatusBadge";
import { WithdrawSectionDialog } from "./WithdrawSectionDialog";

interface SectionEnrollmentsTableProps {
  /** The section whose roster to display. */
  sectionId: string;
}

/**
 * Admin roster table for a selected section.
 * Lists all live section enrollments with status, studentId prefix,
 * and registeredAt date. Provides per-row "Retirar" action for in_progress
 * enrollments and a global "Inscribir alumno" action (requires enrollment.manage).
 */
export function SectionEnrollmentsTable({
  sectionId,
}: SectionEnrollmentsTableProps) {
  const session = useSession();
  const canManage = hasPermission(session, "enrollment.manage");

  const {
    sectionEnrollments,
    isLoading,
    isError,
    refetch,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useSectionEnrollments({ sectionId });

  // Collect student IDs from loaded pages for display-name resolution.
  const studentIds = sectionEnrollments.map((se) => se.studentId);
  const nameMap = useDisplayNames(studentIds);

  const [enrollOpen, setEnrollOpen] = useState(false);
  const [withdrawTarget, setWithdrawTarget] = useState<
    SectionEnrollment | undefined
  >(undefined);

  return (
    <div className="flex flex-col gap-4">
      {canManage && (
        <div className="flex justify-end">
          <Button onClick={() => setEnrollOpen(true)} className="gap-2">
            <Plus className="size-4" aria-hidden />
            Inscribir alumno
          </Button>
        </div>
      )}

      {isLoading && (
        <div
          role="status"
          aria-busy="true"
          aria-label="Cargando inscripciones"
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
            No se pudo cargar la lista de inscripciones.
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
            No hay inscripciones en esta sección.
          </p>
        </div>
      )}

      {!isLoading && !isError && sectionEnrollments.length > 0 && (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Estudiante</TableHead>
                <TableHead>Estado</TableHead>
                <TableHead>Inscrito el</TableHead>
                {canManage && <TableHead className="w-[120px]" />}
              </TableRow>
            </TableHeader>
            <TableBody>
              {sectionEnrollments.map((se) => (
                <SectionEnrollmentRow
                  key={se.id}
                  sectionEnrollment={se}
                  nameMap={nameMap}
                  canManage={canManage}
                  onWithdraw={() => setWithdrawTarget(se)}
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

      {enrollOpen && (
        <EnrollSectionDialog
          open
          sectionId={sectionId}
          onOpenChange={(next) => {
            if (!next) setEnrollOpen(false);
          }}
        />
      )}

      {withdrawTarget && (
        <WithdrawSectionDialog
          open
          onOpenChange={(next) => {
            if (!next) setWithdrawTarget(undefined);
          }}
          sectionEnrollment={withdrawTarget}
        />
      )}
    </div>
  );
}

// --- Row subcomponent ---

interface SectionEnrollmentRowProps {
  sectionEnrollment: SectionEnrollment;
  /** userId → displayName map; falls back to studentId slice when entry absent. */
  nameMap: Map<string, string>;
  canManage: boolean;
  onWithdraw: () => void;
}

function SectionEnrollmentRow({
  sectionEnrollment,
  nameMap,
  canManage,
  onWithdraw,
}: SectionEnrollmentRowProps) {
  const studentDisplay =
    (nameMap.get(sectionEnrollment.studentId) ??
      sectionEnrollment.studentId.slice(0, 8)) ||
    sectionEnrollment.enrollmentId.slice(0, 8);

  const registeredAtDisplay = sectionEnrollment.registeredAt
    ? new Date(sectionEnrollment.registeredAt).toLocaleDateString("es-CL")
    : "—";

  return (
    <TableRow>
      <TableCell className="font-mono text-sm">{studentDisplay}</TableCell>
      <TableCell>
        <SectionEnrollmentStatusBadge status={sectionEnrollment.status} />
      </TableCell>
      <TableCell className="text-muted-foreground text-sm">
        {registeredAtDisplay}
      </TableCell>
      {canManage && (
        <TableCell className="flex gap-2 justify-end">
          {sectionEnrollment.status === "in_progress" && (
            <Button
              variant="ghost"
              size="sm"
              className="text-destructive hover:text-destructive"
              onClick={onWithdraw}
            >
              Retirar
            </Button>
          )}
        </TableCell>
      )}
    </TableRow>
  );
}
