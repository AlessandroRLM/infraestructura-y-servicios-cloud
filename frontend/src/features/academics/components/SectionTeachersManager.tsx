import { LoaderCircle, RefreshCw, UserRound, X } from "lucide-react";
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
import { useUsersList } from "@/features/users";
import type { SectionTeacher } from "@/gen/catalog/v1/catalog_pb";
import { useAssignTeacher } from "../hooks/useAssignTeacher";
import { useRemoveTeacher } from "../hooks/useRemoveTeacher";
import { useSectionTeachers } from "../hooks/useSectionTeachers";
import { mapSectionTeacherError } from "./errorMapping";

interface SectionTeachersManagerProps {
  sectionId: string;
}

export function SectionTeachersManager({
  sectionId,
}: SectionTeachersManagerProps) {
  const {
    sectionTeachers,
    isLoading: isListLoading,
    isError: isListError,
    refetch,
  } = useSectionTeachers(sectionId);

  const { users } = useUsersList("");
  const assign = useAssignTeacher();
  const remove = useRemoveTeacher();

  const [pendingRemove, setPendingRemove] = useState<SectionTeacher | null>(
    null,
  );
  const [comboboxOpen, setComboboxOpen] = useState(false);

  const isPending = assign.isPending || remove.isPending;

  const assignedIds = new Set(sectionTeachers.map((st) => st.teacherId));
  const availableTeachers = users.filter((u) => !assignedIds.has(u.id));

  const handleAssign = async (teacherId: string) => {
    setComboboxOpen(false);
    try {
      await assign.mutateAsync({ sectionId, teacherId });
      toast.success("Docente asignado.");
    } catch (err) {
      toast.error(mapSectionTeacherError(err));
    }
  };

  const handleRemoveConfirm = async (e: React.MouseEvent) => {
    e.preventDefault();
    if (!pendingRemove) return;
    const teacherId = pendingRemove.teacherId;
    try {
      await remove.mutateAsync({ sectionId, teacherId });
      setPendingRemove(null);
      toast.success("Docente quitado.");
    } catch (err) {
      setPendingRemove(null);
      toast.error(mapSectionTeacherError(err));
    }
  };

  if (isListLoading) {
    return (
      <div
        role="status"
        aria-busy="true"
        aria-label="Cargando docentes"
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
            No se pudo cargar los docentes de la sección.
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
    <div className="flex flex-col gap-4 px-4 pb-4">
      {/* Assign combobox */}
      <Popover open={comboboxOpen} onOpenChange={setComboboxOpen}>
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            role="combobox"
            aria-label="Agregar docente"
            aria-expanded={comboboxOpen}
            aria-haspopup="listbox"
            disabled={isPending || availableTeachers.length === 0}
            className="justify-between"
          >
            Agregar docente
          </Button>
        </PopoverTrigger>
        <PopoverContent className="p-0 w-[var(--radix-popover-trigger-width)]">
          <Command>
            <CommandInput placeholder="Buscar docente…" />
            <CommandList>
              <CommandEmpty>No hay docentes disponibles.</CommandEmpty>
              <CommandGroup>
                {availableTeachers.map((u) => (
                  <CommandItem
                    key={u.id}
                    value={u.displayName}
                    onSelect={() => handleAssign(u.id)}
                  >
                    {u.displayName}
                  </CommandItem>
                ))}
              </CommandGroup>
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>

      {/* Assigned teacher list */}
      {sectionTeachers.length === 0 ? (
        <Empty>
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <UserRound />
            </EmptyMedia>
            <EmptyTitle>Sin docentes</EmptyTitle>
            <EmptyDescription>
              Esta sección no tiene docentes asignados todavía.
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : (
        <ul className="flex flex-col gap-2">
          {sectionTeachers.map((st) => {
            const teacher = users.find((u) => u.id === st.teacherId);
            return (
              <li
                key={st.teacherId}
                className="flex items-center justify-between rounded-md border px-3 py-2 text-sm"
              >
                <span className="font-medium">
                  {teacher?.displayName ?? st.teacherId}
                </span>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label={`Quitar ${teacher?.displayName ?? st.teacherId} de la sección`}
                  disabled={isPending}
                  onClick={() => setPendingRemove(st)}
                >
                  <X className="size-4" aria-hidden />
                </Button>
              </li>
            );
          })}
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
            <AlertDialogTitle>¿Quitar docente?</AlertDialogTitle>
            <AlertDialogDescription>
              ¿Quitar{" "}
              {users.find((u) => u.id === pendingRemove?.teacherId)
                ?.displayName ?? pendingRemove?.teacherId}{" "}
              de la sección? Podrás volver a asignarlo.
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
