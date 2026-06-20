import { LoaderCircle } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { StudentPicker } from "@/features/reports/components/StudentPicker";
import { useCreateEnrollment } from "../hooks/useCreateEnrollment";
import { mapCreateEnrollmentError } from "../hooks/errorMapping";
import { ProgramPicker } from "./ProgramPicker";

interface CreateEnrollmentDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/**
 * Modal dialog for creating a new enrollment.
 * - StudentPicker: reused from features/reports (IamService.listUsers).
 * - ProgramPicker: feature-local (CatalogService.listPrograms).
 * - Year input: local text buffer (untypeable-year fix) — emits undefined for
 *   partial/out-of-range values, valid int32 for complete valid years.
 * - Submit disabled until studentId && programId && valid year.
 * - Domain errors (AlreadyExists/InvalidArgument) shown inline.
 * - Transport errors shown as toast.
 */
export function CreateEnrollmentDialog({
  open,
  onOpenChange,
}: CreateEnrollmentDialogProps) {
  const createMutation = useCreateEnrollment();

  const [studentId, setStudentId] = useState("");
  const [programId, setProgramId] = useState("");
  const [year, setYear] = useState<number | undefined>(undefined);
  const [yearText, setYearText] = useState("");
  const [inlineError, setInlineError] = useState<string | null>(null);

  // Reset form state when the dialog is closed.
  const handleOpenChange = (next: boolean) => {
    if (!next) {
      setStudentId("");
      setProgramId("");
      setYear(undefined);
      setYearText("");
      setInlineError(null);
    }
    onOpenChange(next);
  };

  // Sync external year prop → local text only when year is set (navigation sync pattern
  // from ProgramYearPicker: do NOT clear on transition to undefined to avoid wiping
  // mid-type input).
  useEffect(() => {
    if (year != null) setYearText(String(year));
  }, [year]);

  const handleYearChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const raw = e.target.value;
    setYearText(raw);
    const trimmed = raw.trim();
    if (trimmed === "") {
      setYear(undefined);
      return;
    }
    const parsed = Number.parseInt(trimmed, 10);
    if (!Number.isNaN(parsed) && parsed >= 2000 && parsed <= 2100) {
      setYear(parsed);
    } else {
      setYear(undefined);
    }
  };

  const canSubmit =
    studentId !== "" && programId !== "" && year != null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSubmit) return;
    setInlineError(null);
    try {
      await createMutation.mutateAsync({
        studentId,
        programId,
        year,
      });
      handleOpenChange(false);
      toast.success("Matrícula creada");
    } catch (err) {
      const message = mapCreateEnrollmentError(err);
      if (message) {
        setInlineError(message);
      } else {
        toast.error("No se pudo crear la matrícula. Inténtalo de nuevo.");
      }
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Crear matrícula</DialogTitle>
          <DialogDescription>
            Registra una matrícula anual para un estudiante en un programa
            académico.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="enrollment-student">Estudiante</Label>
            <StudentPicker value={studentId} onChange={setStudentId} />
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="enrollment-program">Programa</Label>
            <ProgramPicker value={programId} onChange={setProgramId} />
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="enrollment-year">Año académico</Label>
            <Input
              id="enrollment-year"
              type="number"
              min={2000}
              max={2100}
              placeholder="Ej. 2026"
              value={yearText}
              onChange={handleYearChange}
              className="max-w-[160px]"
            />
          </div>

          {inlineError && (
            <p role="alert" className="text-destructive text-sm">
              {inlineError}
            </p>
          )}

          <DialogFooter>
            <Button
              type="submit"
              disabled={!canSubmit || createMutation.isPending}
            >
              {createMutation.isPending ? (
                <>
                  <LoaderCircle
                    className="size-4 animate-spin"
                    data-icon="inline-start"
                    aria-hidden
                  />
                  Creando…
                </>
              ) : (
                "Crear matrícula"
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
