import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import type { Section } from "@/gen/catalog/v1/catalog_pb";
import { SectionTeachersManager } from "./SectionTeachersManager";

interface SectionTeachersSheetProps {
  section: Section | undefined;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function SectionTeachersSheet({
  section,
  open,
  onOpenChange,
}: SectionTeachersSheetProps) {
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full sm:max-w-md overflow-y-auto">
        <SheetHeader>
          <SheetTitle>
            Docentes de la sección (cap. {section?.seatCapacity})
          </SheetTitle>
          <SheetDescription className="sr-only">
            Gestión de docentes de la sección
          </SheetDescription>
        </SheetHeader>
        {section && <SectionTeachersManager sectionId={section.id} />}
      </SheetContent>
    </Sheet>
  );
}
