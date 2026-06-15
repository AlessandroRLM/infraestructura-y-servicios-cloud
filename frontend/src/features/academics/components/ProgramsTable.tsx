import {
  BookOpen,
  Loader2,
  MoreHorizontal,
  Pencil,
  Plus,
  RefreshCw,
  Trash2,
} from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { PageSizeSelector, SearchInput } from "@/core/components";
import { hasPermission, useSession } from "@/features/auth";
import type { Program } from "@/gen/catalog/v1/catalog_pb";
import { Route } from "@/routes/_authenticated/admin/academics";
import { SEARCH_DEBOUNCE_MS } from "../constants";
import { usePrograms } from "../hooks/usePrograms";
import { DeleteProgramDialog } from "./DeleteProgramDialog";
import { ProgramCoursesSheet } from "./ProgramCoursesSheet";
import { ProgramDialog } from "./ProgramDialog";

interface ProgramsTableProps {
  onCreateClick?: () => void;
}

export function ProgramsTable({ onCreateClick }: ProgramsTableProps) {
  const session = useSession();
  const canManage = hasPermission(session, "catalog.manage");

  const { q, pageSize } = Route.useSearch();
  const navigate = Route.useNavigate();

  const {
    programs,
    isLoading,
    isError,
    refetch,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = usePrograms(q, pageSize);

  const [editProgram, setEditProgram] = useState<Program | undefined>(
    undefined,
  );
  const [deleteProgram, setDeleteProgram] = useState<Program | undefined>(
    undefined,
  );
  const [coursesProgram, setCoursesProgram] = useState<Program | undefined>(
    undefined,
  );

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-3">
        <SearchInput
          value={q}
          onChange={(v) => navigate({ search: (prev) => ({ ...prev, q: v }) })}
          debounceMs={SEARCH_DEBOUNCE_MS}
          placeholder="Buscar por código o nombre..."
          className="max-w-sm"
        />
        <PageSizeSelector
          value={pageSize}
          onChange={(n) =>
            navigate({ search: (prev) => ({ ...prev, pageSize: n }) })
          }
        />
      </div>

      {isLoading && (
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
      )}

      {!isLoading && isError && (
        <div
          className="rounded-md border border-destructive/50 p-4"
          role="alert"
        >
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
      )}

      {!isLoading && !isError && programs.length === 0 && (
        <div className="flex flex-col items-center justify-center gap-4 rounded-md border border-dashed p-12 text-center">
          <p className="text-muted-foreground text-sm">
            Todavía no hay carreras
          </p>
          {canManage && (
            <Button onClick={onCreateClick} className="gap-2">
              <Plus className="size-4" aria-hidden />
              Crear carrera
            </Button>
          )}
        </div>
      )}

      {!isLoading && !isError && programs.length > 0 && (
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
                    <TableCell className="flex justify-end">
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button
                            variant="ghost"
                            size="icon"
                            aria-label={`Acciones ${program.code}`}
                          >
                            <MoreHorizontal aria-hidden />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuGroup>
                            <DropdownMenuItem
                              onSelect={() => setEditProgram(program)}
                            >
                              <Pencil data-icon="inline-start" />
                              Editar
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              onSelect={() => setCoursesProgram(program)}
                            >
                              <BookOpen data-icon="inline-start" />
                              Asignaturas
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              variant="destructive"
                              onSelect={() => setDeleteProgram(program)}
                            >
                              <Trash2 data-icon="inline-start" />
                              Eliminar
                            </DropdownMenuItem>
                          </DropdownMenuGroup>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                  )}
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
                toast.error("No se pudieron cargar más carreras.");
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

      <ProgramCoursesSheet
        program={coursesProgram}
        open={!!coursesProgram}
        onOpenChange={(open) => {
          if (!open) setCoursesProgram(undefined);
        }}
      />
    </div>
  );
}
