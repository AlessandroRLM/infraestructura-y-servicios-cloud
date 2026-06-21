import { Plus } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { hasPermission, useSession } from "@/features/auth";
import { ProgramQuotaDialog } from "./ProgramQuotaDialog";
import { ProgramQuotasTable } from "./ProgramQuotasTable";

interface ProgramQuotasManagerProps {
  /** UUID of the program whose quotas are managed. Must be a non-empty string. */
  programId: string;
}

/**
 * Self-contained manager for the quotas of a single program.
 * Renders a "Crear cupo" button (gated by catalog.manage) and delegates
 * listing, editing, and deleting to ProgramQuotasTable.
 */
export function ProgramQuotasManager({ programId }: ProgramQuotasManagerProps) {
  const session = useSession();
  const canManage = hasPermission(session, "catalog.manage");
  const [createOpen, setCreateOpen] = useState(false);

  return (
    <div className="flex flex-col gap-4 px-4 pb-4">
      {canManage && (
        <Button
          variant="outline"
          className="self-start gap-2"
          onClick={() => setCreateOpen(true)}
        >
          <Plus className="size-4" aria-hidden />
          Crear cupo
        </Button>
      )}

      <ProgramQuotasTable
        programId={programId}
        onCreateClick={() => setCreateOpen(true)}
      />

      <ProgramQuotaDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        programId={programId}
      />
    </div>
  );
}
