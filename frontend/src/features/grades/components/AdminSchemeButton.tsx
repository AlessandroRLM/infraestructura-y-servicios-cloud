import { Settings } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { SchemeManagementView } from "./SchemeManagementView";

interface AdminSchemeButtonProps {
  /** Course ID pre-selected when the dialog opens. Reserved for future use. */
  courseId: string;
  /** Button variant; defaults to "default". */
  variant?: "default" | "outline" | "ghost";
}

/**
 * Admin-only button that opens SchemeManagementView in a dialog.
 * Rendered inside GradeRecordingGrid top-right for users with grades.override.
 * Also rendered in the empty-scheme state as a secondary call-to-action.
 *
 * The button is never rendered for non-admin users — the caller is responsible
 * for the grades.override gate.
 */
export function AdminSchemeButton({
  courseId: _courseId,
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
          <SchemeManagementView />
        </DialogContent>
      </Dialog>
    </>
  );
}
