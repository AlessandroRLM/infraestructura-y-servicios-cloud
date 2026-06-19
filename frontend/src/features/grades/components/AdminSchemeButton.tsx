import { Settings } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { SchemeManagementView } from "./SchemeManagementView";

interface AdminSchemeButtonProps {
  /** Course ID forwarded to SchemeManagementView as initialCourseId. */
  courseId: string;
  /** Button variant; defaults to "default". */
  variant?: "default" | "outline" | "ghost";
}

/**
 * Admin-only button that opens SchemeManagementView in a dialog.
 * Rendered inside GradeRecordingGrid top-right for users with grades.override.
 * Also rendered in the empty-scheme state as a secondary call-to-action.
 *
 * Forwards courseId as initialCourseId so the picker opens pre-scoped to the
 * section's course — no redundant search needed.
 *
 * The button is never rendered for non-admin users — the caller is responsible
 * for the grades.override gate.
 */
export function AdminSchemeButton({
  courseId,
  variant = "default",
}: AdminSchemeButtonProps) {
  const [open, setOpen] = useState(false);

  return (
    <>
      <Button
        variant={variant}
        size="sm"
        className="gap-2"
        onClick={() => setOpen(true)}
      >
        <Settings className="size-4" data-icon="inline-start" aria-hidden />
        Administrar Notas
      </Button>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-3xl max-h-[90vh] overflow-y-auto">
          <DialogTitle className="sr-only">
            Administrar esquema de evaluación
          </DialogTitle>
          <SchemeManagementView initialCourseId={courseId} />
        </DialogContent>
      </Dialog>
    </>
  );
}
