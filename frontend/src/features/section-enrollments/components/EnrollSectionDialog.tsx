import { ChevronsUpDown, LoaderCircle } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
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
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { mapEnrollSectionError } from "../hooks/errorMapping";
import { useDisplayNames } from "../hooks/useDisplayNames";
import { useEnrollmentsForPicker } from "../hooks/useEnrollmentsForPicker";
import { useEnrollSection } from "../hooks/useEnrollSection";

interface EnrollSectionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The section to enroll into. */
  sectionId: string;
}

/**
 * Modal dialog for enrolling a student (via their paid enrollment) into a section.
 * - EnrollmentPicker: Popover + Command backed by EnrollmentService.listEnrollments (paid only).
 * - Submit disabled until an enrollment is selected.
 * - Domain errors (FailedPrecondition / AlreadyExists) shown inline.
 * - Transport errors shown as toast.
 * - Reset state on close.
 */
export function EnrollSectionDialog({
  open,
  onOpenChange,
  sectionId,
}: EnrollSectionDialogProps) {
  const enrollMutation = useEnrollSection();
  const [enrollmentId, setEnrollmentId] = useState("");
  const [enrollmentLabel, setEnrollmentLabel] = useState<string | null>(null);
  const [inlineError, setInlineError] = useState<string | null>(null);
  const [pickerOpen, setPickerOpen] = useState(false);
  const [search, setSearch] = useState("");

  const { enrollments, isLoading } = useEnrollmentsForPicker();

  // Resolve display names for all students in the picker.
  const pickerStudentIds = enrollments.map((e) => e.studentId);
  const nameMap = useDisplayNames(pickerStudentIds);

  const handleOpenChange = (next: boolean) => {
    if (!next) {
      setEnrollmentId("");
      setEnrollmentLabel(null);
      setInlineError(null);
      setSearch("");
    }
    onOpenChange(next);
  };

  const handlePickerOpenChange = (next: boolean) => {
    if (!next) setSearch("");
    setPickerOpen(next);
  };

  const handleSelect = (id: string) => {
    const picked = enrollments.find((e) => e.id === id);
    if (picked) {
      const student =
        nameMap.get(picked.studentId) ?? picked.studentId.slice(0, 8);
      const program =
        (picked.programName ?? "").trim() || picked.programId.slice(0, 8);
      setEnrollmentLabel(`${student} — ${program} (${picked.year})`);
    } else {
      setEnrollmentLabel(null);
    }
    setEnrollmentId(id);
    setPickerOpen(false);
    setSearch("");
  };

  const canSubmit = enrollmentId !== "";

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSubmit) return;
    setInlineError(null);
    try {
      await enrollMutation.mutateAsync({ enrollmentId, sectionId });
      handleOpenChange(false);
      toast.success("Alumno inscrito en la sección");
    } catch (err) {
      const kind = mapEnrollSectionError(err);
      if (kind === "precondition") {
        setInlineError(
          "No se puede inscribir: la sección está llena, el período está cerrado, o la matrícula no cumple las condiciones.",
        );
      } else if (kind === "saturated") {
        setInlineError(
          "El sistema está procesando muchas inscripciones simultáneas. Inténtalo de nuevo en unos segundos.",
        );
      } else if (kind === "already_enrolled") {
        setInlineError("El estudiante ya está inscrito en esta sección.");
      } else {
        toast.error("No se pudo inscribir al alumno. Inténtalo de nuevo.");
      }
    }
  };

  const pickerLabel =
    enrollmentId === ""
      ? "Seleccionar matrícula"
      : (enrollmentLabel ?? "Seleccionar matrícula");

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Inscribir alumno</DialogTitle>
          <DialogDescription>
            Selecciona la matrícula anual del estudiante para inscribirlo en
            esta sección. Solo se muestran matrículas pagadas.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label>Matrícula del estudiante</Label>
            <Popover open={pickerOpen} onOpenChange={handlePickerOpenChange}>
              <PopoverTrigger asChild>
                <Button
                  type="button"
                  variant="outline"
                  role="combobox"
                  aria-label="Seleccionar matrícula"
                  aria-expanded={pickerOpen}
                  aria-haspopup="listbox"
                  className="w-full justify-between"
                >
                  <span className="truncate">{pickerLabel}</span>
                  <ChevronsUpDown
                    className="size-4 shrink-0 text-muted-foreground"
                    aria-hidden
                  />
                </Button>
              </PopoverTrigger>
              <PopoverContent className="p-0 w-[var(--radix-popover-trigger-width)]">
                <Command shouldFilter={false}>
                  <CommandInput
                    placeholder="Buscar por estudiante o programa…"
                    value={search}
                    onValueChange={setSearch}
                  />
                  <CommandList>
                    {isLoading ? (
                      <div className="py-6 text-center text-sm text-muted-foreground">
                        Cargando…
                      </div>
                    ) : (
                      <>
                        <CommandEmpty>
                          No se encontraron matrículas pagadas.
                        </CommandEmpty>
                        <CommandGroup>
                          {enrollments.map((e) => {
                            const student =
                              nameMap.get(e.studentId) ??
                              e.studentId.slice(0, 8);
                            const program =
                              (e.programName ?? "").trim() ||
                              e.programId.slice(0, 8);
                            return (
                              <CommandItem
                                key={e.id}
                                value={e.id}
                                onSelect={() => handleSelect(e.id)}
                              >
                                {student} — {program} ({e.year})
                              </CommandItem>
                            );
                          })}
                        </CommandGroup>
                      </>
                    )}
                  </CommandList>
                </Command>
              </PopoverContent>
            </Popover>
          </div>

          {inlineError && (
            <p role="alert" className="text-destructive text-sm">
              {inlineError}
            </p>
          )}

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => handleOpenChange(false)}
            >
              Cancelar
            </Button>
            <Button
              type="submit"
              disabled={!canSubmit || enrollMutation.isPending}
            >
              {enrollMutation.isPending ? (
                <>
                  <LoaderCircle
                    className="size-4 animate-spin"
                    data-icon="inline-start"
                    aria-hidden
                  />
                  Inscribiendo…
                </>
              ) : (
                "Inscribir alumno"
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
