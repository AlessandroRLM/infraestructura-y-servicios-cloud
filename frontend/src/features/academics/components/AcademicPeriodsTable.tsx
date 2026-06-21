import { Pencil, Plus, RefreshCw, Trash2 } from "lucide-react";
import { useState } from "react";
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
import { academicPeriodLabel, useAcademicPeriods } from "@/core/catalog";
import { hasPermission, useSession } from "@/features/auth";
import type { AcademicPeriod } from "@/gen/catalog/v1/catalog_pb";
import { AcademicPeriodDialog } from "./AcademicPeriodDialog";
import { DeleteAcademicPeriodDialog } from "./DeleteAcademicPeriodDialog";

interface AcademicPeriodsTableProps {
  onCreateClick?: () => void;
}

export function AcademicPeriodsTable({
  onCreateClick,
}: AcademicPeriodsTableProps) {
  const session = useSession();
  const canManage = hasPermission(session, "catalog.manage");

  const { periods, isLoading, isError, refetch } = useAcademicPeriods();

  const [editPeriod, setEditPeriod] = useState<AcademicPeriod | undefined>(
    undefined,
  );
  const [deletePeriod, setDeletePeriod] = useState<AcademicPeriod | undefined>(
    undefined,
  );

  return (
    <div className="flex flex-col gap-4">
      {isLoading && (
        <div
          role="status"
          className="space-y-2"
          aria-busy="true"
          aria-label="Cargando períodos"
        >
          {Array.from({ length: 4 }).map((_, i) => (
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
            No se pudo cargar la lista de períodos.
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

      {!isLoading && !isError && periods.length === 0 && (
        <div className="flex flex-col items-center justify-center gap-4 rounded-md border border-dashed p-12 text-center">
          <p className="text-muted-foreground text-sm">
            Todavía no hay períodos académicos
          </p>
          {canManage && (
            <Button onClick={onCreateClick} className="gap-2">
              <Plus className="size-4" aria-hidden />
              Crear período
            </Button>
          )}
        </div>
      )}

      {!isLoading && !isError && periods.length > 0 && (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Año</TableHead>
                <TableHead>Semestre</TableHead>
                <TableHead>Inicio</TableHead>
                <TableHead>Término</TableHead>
                {canManage && <TableHead className="w-[120px]" />}
              </TableRow>
            </TableHeader>
            <TableBody>
              {periods.map((period) => (
                <TableRow key={period.id}>
                  <TableCell className="font-medium">{period.year}</TableCell>
                  <TableCell>{period.term}</TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {period.startDate || "—"}
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {period.endDate || "—"}
                  </TableCell>
                  {canManage && (
                    <TableCell className="flex gap-2 justify-end">
                      <Button
                        variant="ghost"
                        size="sm"
                        className="gap-1"
                        onClick={() => setEditPeriod(period)}
                        aria-label={`Editar período ${academicPeriodLabel(period.year, period.term)}`}
                      >
                        <Pencil className="size-4" aria-hidden />
                        Editar
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="gap-1 text-destructive hover:text-destructive"
                        onClick={() => setDeletePeriod(period)}
                        aria-label={`Eliminar período ${academicPeriodLabel(period.year, period.term)}`}
                      >
                        <Trash2 className="size-4" aria-hidden />
                        Eliminar
                      </Button>
                    </TableCell>
                  )}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {editPeriod && (
        <AcademicPeriodDialog
          open={true}
          onOpenChange={(open) => {
            if (!open) setEditPeriod(undefined);
          }}
          period={editPeriod}
        />
      )}

      {deletePeriod && (
        <DeleteAcademicPeriodDialog
          open={true}
          onOpenChange={(open) => {
            if (!open) setDeletePeriod(undefined);
          }}
          period={deletePeriod}
        />
      )}
    </div>
  );
}
