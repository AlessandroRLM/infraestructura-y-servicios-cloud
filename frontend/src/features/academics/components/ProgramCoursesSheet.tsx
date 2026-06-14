import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import type { Program } from "@/gen/catalog/v1/catalog_pb";
import { ProgramCoursesManager } from "./ProgramCoursesManager";

interface ProgramCoursesSheetProps {
  program: Program | undefined;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ProgramCoursesSheet({
  program,
  open,
  onOpenChange,
}: ProgramCoursesSheetProps) {
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full sm:max-w-md overflow-y-auto">
        <SheetHeader>
          <SheetTitle>Asignaturas de {program?.name}</SheetTitle>
          <SheetDescription className="sr-only">
            Gestión de asignaturas de la carrera
          </SheetDescription>
        </SheetHeader>
        {program && <ProgramCoursesManager programId={program.id} />}
      </SheetContent>
    </Sheet>
  );
}
