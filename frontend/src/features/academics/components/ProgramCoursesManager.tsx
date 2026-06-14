import { BookOpen, LoaderCircle, RefreshCw, X } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Skeleton } from "@/components/ui/skeleton";
import type { ProgramCourse } from "@/gen/catalog/v1/catalog_pb";
import { useAddCourseToProgram } from "../hooks/useAddCourseToProgram";
import { useCourses } from "../hooks/useCourses";
import { useProgramCourses } from "../hooks/useProgramCourses";
import { useRemoveCourseFromProgram } from "../hooks/useRemoveCourseFromProgram";
import { mapProgramCourseError } from "./errorMapping";

interface ProgramCoursesManagerProps {
  programId: string;
}

export function ProgramCoursesManager({
  programId,
}: ProgramCoursesManagerProps) {
  const {
    programCourses,
    isPending: isListPending,
    isError: isListError,
    refetch,
  } = useProgramCourses(programId);
  const { courses } = useCourses();
  const add = useAddCourseToProgram(programId);
  const remove = useRemoveCourseFromProgram(programId);

  const [pendingRemove, setPendingRemove] = useState<ProgramCourse | null>(
    null,
  );
  const [comboboxOpen, setComboboxOpen] = useState(false);

  const isPending = add.isPending || remove.isPending;

  const associatedIds = new Set(programCourses.map((pc) => pc.courseId));
  const availableCourses = courses.filter((c) => !associatedIds.has(c.id));

  const handleAdd = async (courseId: string) => {
    setComboboxOpen(false);
    try {
      await add.mutateAsync({ programId, courseId });
      toast.success("Asignatura agregada.");
    } catch (err) {
      toast.error(mapProgramCourseError(err));
    }
  };

  const handleRemoveConfirm = async (e: React.MouseEvent) => {
    e.preventDefault();
    if (!pendingRemove) return;
    const courseId = pendingRemove.courseId;
    try {
      await remove.mutateAsync({ programId, courseId });
      setPendingRemove(null);
      toast.success("Asignatura quitada.");
    } catch (err) {
      setPendingRemove(null);
      toast.error(mapProgramCourseError(err));
    }
  };

  if (isListPending) {
    return (
      <div
        role="status"
        aria-busy="true"
        aria-label="Cargando asignaturas"
        className="flex flex-col gap-2 p-4"
      >
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-10 w-full" />
        ))}
      </div>
    );
  }

  if (isListError) {
    return (
      <div className="flex flex-col gap-4 p-4">
        <div
          className="rounded-md border border-destructive/50 p-4"
          role="alert"
        >
          <p className="text-destructive text-sm font-medium">
            No se pudo cargar las asignaturas de la carrera.
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
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4 p-4">
      {/* Add Combobox */}
      <Popover open={comboboxOpen} onOpenChange={setComboboxOpen}>
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            role="combobox"
            aria-label="Agregar asignatura"
            aria-expanded={comboboxOpen}
            aria-haspopup="listbox"
            disabled={isPending || availableCourses.length === 0}
            className="justify-between"
          >
            Agregar asignatura
          </Button>
        </PopoverTrigger>
        <PopoverContent className="p-0 w-[var(--radix-popover-trigger-width)]">
          <Command>
            <CommandInput placeholder="Buscar asignatura…" />
            <CommandList>
              <CommandEmpty>No hay asignaturas disponibles.</CommandEmpty>
              <CommandGroup>
                {availableCourses.map((c) => (
                  <CommandItem
                    key={c.id}
                    value={`${c.code} ${c.name}`}
                    onSelect={() => handleAdd(c.id)}
                  >
                    {c.code} — {c.name}
                  </CommandItem>
                ))}
              </CommandGroup>
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>

      {/* Associated course list */}
      {programCourses.length === 0 ? (
        <Empty>
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <BookOpen />
            </EmptyMedia>
            <EmptyTitle>Sin asignaturas</EmptyTitle>
            <EmptyDescription>
              Esta carrera no tiene asignaturas todavía.
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : (
        <ul className="flex flex-col gap-2">
          {programCourses.map((pc) => (
            <li
              key={pc.courseId}
              className="flex items-center justify-between rounded-md border px-3 py-2 text-sm"
            >
              <div className="flex items-center gap-2 min-w-0">
                <span className="font-medium">
                  {pc.course?.code ?? pc.courseId}
                </span>
                <span className="truncate text-muted-foreground">
                  {pc.course?.name ?? pc.courseId}
                </span>
                {pc.course?.credits != null && (
                  <Badge variant="secondary">
                    {pc.course.credits} créditos
                  </Badge>
                )}
              </div>
              <Button
                variant="ghost"
                size="icon-sm"
                aria-label={`Quitar ${pc.course?.code ?? pc.courseId} de la carrera`}
                disabled={isPending}
                onClick={() => setPendingRemove(pc)}
              >
                <X className="size-4" aria-hidden />
              </Button>
            </li>
          ))}
        </ul>
      )}

      {/* Remove confirmation AlertDialog */}
      <AlertDialog
        open={pendingRemove !== null}
        onOpenChange={(open) => {
          if (!open) setPendingRemove(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>¿Quitar asignatura?</AlertDialogTitle>
            <AlertDialogDescription>
              ¿Quitar {pendingRemove?.course?.code ?? pendingRemove?.courseId}{" "}
              de la carrera? Podrás volver a agregarla.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel
              disabled={remove.isPending}
              onClick={() => setPendingRemove(null)}
            >
              Cancelar
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={handleRemoveConfirm}
              disabled={remove.isPending}
              className="gap-2"
            >
              {remove.isPending ? (
                <>
                  <LoaderCircle className="size-4 animate-spin" aria-hidden />
                  Quitando…
                </>
              ) : (
                "Quitar"
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
