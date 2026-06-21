import { useState } from "react";
import { Button } from "@/components/ui/button";
import { CreateEnrollmentDialog } from "./CreateEnrollmentDialog";
import { EnrollmentsTable } from "./EnrollmentsTable";

/**
 * Admin enrollments page.
 * Thin container: page header + Crear button + EnrollmentsTable.
 * All data-fetching and URL state live in EnrollmentsTable (ADR-5).
 */
export function EnrollmentsPage() {
  const [createOpen, setCreateOpen] = useState(false);

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <h1 className="font-semibold text-2xl tracking-tight">Matrículas</h1>
        <Button onClick={() => setCreateOpen(true)}>Crear matrícula</Button>
      </div>

      <EnrollmentsTable />

      <CreateEnrollmentDialog open={createOpen} onOpenChange={setCreateOpen} />
    </div>
  );
}
