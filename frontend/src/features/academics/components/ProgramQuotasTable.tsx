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
import { hasPermission, useSession } from "@/features/auth";
import type { ProgramQuota } from "@/gen/catalog/v1/catalog_pb";
import { useProgramQuotas } from "../hooks/useProgramQuotas";
import { DeleteProgramQuotaDialog } from "./DeleteProgramQuotaDialog";
import { ProgramQuotaDialog } from "./ProgramQuotaDialog";

interface ProgramQuotasTableProps {
  /** UUID of the currently-selected program. Quotas are fetched only when non-empty. */
  programId: string;
  onCreateClick?: () => void;
}

export function ProgramQuotasTable({
  programId,
  onCreateClick,
}: ProgramQuotasTableProps) {
  const session = useSession();
  const canManage = hasPermission(session, "catalog.manage");

  const { programQuotas, isLoading, isError, refetch } =
    useProgramQuotas(programId);

  const [editQuota, setEditQuota] = useState<ProgramQuota | undefined>(
    undefined,
  );
  const [deleteQuota, setDeleteQuota] = useState<ProgramQuota | undefined>(
    undefined,
  );

  return (
    <div className="flex flex-col gap-4">
      {isLoading && (
        <div
          role="status"
          className="space-y-2"
          aria-busy="true"
          aria-label="Cargando cupos"
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
            No se pudo cargar la lista de cupos.
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

      {!isLoading && !isError && programQuotas.length === 0 && (
        <div className="flex flex-col items-center justify-center gap-4 rounded-md border border-dashed p-12 text-center">
          <p className="text-muted-foreground text-sm">Todavía no hay cupos</p>
          {canManage && (
            <Button onClick={onCreateClick} className="gap-2">
              <Plus className="size-4" aria-hidden />
              Crear cupo
            </Button>
          )}
        </div>
      )}

      {!isLoading && !isError && programQuotas.length > 0 && (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Año</TableHead>
                <TableHead>Cupo</TableHead>
                {canManage && <TableHead className="w-[160px]" />}
              </TableRow>
            </TableHeader>
            <TableBody>
              {programQuotas.map((quota) => (
                <TableRow key={quota.id}>
                  <TableCell className="text-muted-foreground text-sm">
                    {quota.year}
                  </TableCell>
                  <TableCell>{quota.admissionQuota}</TableCell>
                  {canManage && (
                    <TableCell className="flex gap-2 justify-end">
                      <Button
                        variant="ghost"
                        size="sm"
                        className="gap-1"
                        onClick={() => setEditQuota(quota)}
                        aria-label={`Editar cupo ${quota.id}`}
                      >
                        <Pencil className="size-4" aria-hidden />
                        Editar
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="gap-1 text-destructive hover:text-destructive"
                        onClick={() => setDeleteQuota(quota)}
                        aria-label={`Eliminar cupo ${quota.id}`}
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

      {editQuota && (
        <ProgramQuotaDialog
          open={true}
          onOpenChange={(open) => {
            if (!open) setEditQuota(undefined);
          }}
          quota={editQuota}
          programId={programId}
        />
      )}

      {deleteQuota && (
        <DeleteProgramQuotaDialog
          open={true}
          onOpenChange={(open) => {
            if (!open) setDeleteQuota(undefined);
          }}
          quota={deleteQuota}
        />
      )}
    </div>
  );
}
