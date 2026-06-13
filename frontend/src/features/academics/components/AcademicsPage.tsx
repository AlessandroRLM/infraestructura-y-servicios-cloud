import { Plus } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { hasPermission, useSession } from "@/features/auth";
import { ProgramDialog } from "./ProgramDialog";
import { ProgramsTable } from "./ProgramsTable";

export function AcademicsPage() {
  const session = useSession();
  const canManage = hasPermission(session, "catalog.manage");
  const [createDialogOpen, setCreateDialogOpen] = useState(false);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="font-semibold text-2xl tracking-tight">Académico</h1>
        {canManage && (
          <Button onClick={() => setCreateDialogOpen(true)} className="gap-2">
            <Plus className="size-4" aria-hidden />
            Crear programa
          </Button>
        )}
      </div>

      <ProgramsTable onCreateClick={() => setCreateDialogOpen(true)} />

      <ProgramDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
      />
    </div>
  );
}
