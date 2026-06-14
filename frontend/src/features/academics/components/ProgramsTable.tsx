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
import type { Program } from "@/gen/catalog/v1/catalog_pb";
import { usePrograms } from "../hooks/usePrograms";
import { DeleteProgramDialog } from "./DeleteProgramDialog";
import { ProgramDialog } from "./ProgramDialog";

interface ProgramsTableProps {
  onCreateClick?: () => void;
}

export function ProgramsTable({ onCreateClick }: ProgramsTableProps) {
  const session = useSession();
  const canManage = hasPermission(session, "catalog.manage");
  const { programs, isLoading, isError, refetch } = usePrograms();

  const [editProgram, setEditProgram] = useState<Program | undefined>(
    undefined,
  );
  const [deleteProgram, setDeleteProgram] = useState<Program | undefined>(
    undefined,
  );

  if (isLoading) {
    return (
      <div
        role="status"
        className="space-y-2"
        aria-busy="true"
        aria-label="Cargando carreras"
      >
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-10 w-full" />
        ))}
      </div>
    );
  }

  if (isError) {
    return (
      <div className="rounded-md border border-destructive/50 p-4" role="alert">
        <p className="text-destructive text-sm font-medium">
          No se pudo cargar la lista de carreras.
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
    );
  }

  if (programs.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-4 rounded-md border border-dashed p-12 text-center">
        <p className="text-muted-foreground text-sm">Todavía no hay carreras</p>
        {canManage && (
          <Button onClick={onCreateClick} className="gap-2">
            <Plus className="size-4" aria-hidden />
            Crear carrera
          </Button>
        )}
      </div>
    );
  }

  return (
    <>
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Código</TableHead>
              <TableHead>Nombre</TableHead>
              <TableHead>Creado</TableHead>
              {canManage && <TableHead className="w-[120px]" />}
            </TableRow>
          </TableHeader>
          <TableBody>
            {programs.map((program) => (
              <TableRow key={program.id}>
                <TableCell className="font-medium">{program.code}</TableCell>
                <TableCell>{program.name}</TableCell>
                <TableCell className="text-muted-foreground text-sm">
                  {program.createdAt
                    ? new Date(program.createdAt).toLocaleDateString("es-CL")
                    : "—"}
                </TableCell>
                {canManage && (
                  <TableCell className="flex gap-2 justify-end">
                    <Button
                      variant="ghost"
                      size="sm"
                      className="gap-1"
                      onClick={() => setEditProgram(program)}
                      aria-label={`Editar ${program.code}`}
                    >
                      <Pencil className="size-4" aria-hidden />
                      Editar
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="gap-1 text-destructive hover:text-destructive"
                      onClick={() => setDeleteProgram(program)}
                      aria-label={`Eliminar ${program.code}`}
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

      {editProgram && (
        <ProgramDialog
          open={true}
          onOpenChange={(open) => {
            if (!open) setEditProgram(undefined);
          }}
          program={editProgram}
        />
      )}

      {deleteProgram && (
        <DeleteProgramDialog
          open={true}
          onOpenChange={(open) => {
            if (!open) setDeleteProgram(undefined);
          }}
          program={deleteProgram}
        />
      )}
    </>
  );
}
