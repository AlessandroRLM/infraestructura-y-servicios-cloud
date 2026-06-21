import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import type { Program } from "@/gen/catalog/v1/catalog_pb";
import { ProgramCoursesManager } from "./ProgramCoursesManager";
import { ProgramQuotasManager } from "./ProgramQuotasManager";

interface ProgramCoursesSheetProps {
  program: Program | undefined;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/**
 * Side Sheet for managing all per-program resources.
 * Tabs: Asignaturas (courses) and Cupos (quotas).
 * The programId is always in context — no picker needed inside.
 */
export function ProgramCoursesSheet({
  program,
  open,
  onOpenChange,
}: ProgramCoursesSheetProps) {
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full sm:max-w-lg overflow-y-auto">
        <SheetHeader>
          <SheetTitle>Gestionar {program?.name}</SheetTitle>
          <SheetDescription className="sr-only">
            Gestión de asignaturas y cupos de la carrera
          </SheetDescription>
        </SheetHeader>

        {program && (
          <Tabs defaultValue="courses" className="mt-4">
            <TabsList className="mx-4">
              <TabsTrigger value="courses">Asignaturas</TabsTrigger>
              <TabsTrigger value="quotas">Cupos</TabsTrigger>
            </TabsList>
            <TabsContent value="courses">
              <ProgramCoursesManager programId={program.id} />
            </TabsContent>
            <TabsContent value="quotas">
              <ProgramQuotasManager programId={program.id} />
            </TabsContent>
          </Tabs>
        )}
      </SheetContent>
    </Sheet>
  );
}
