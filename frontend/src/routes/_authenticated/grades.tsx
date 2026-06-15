import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";
import { requireAnyPermission } from "@/features/auth";
import { GradesPage } from "@/features/grades";

const gradesSearchSchema = z.object({
  /** UUID of the academic period to filter by. Empty string means no filter. */
  period: z.string().default("").catch(""),
  /** UUID of the program (carrera) to filter by. Empty string means no filter. */
  program: z.string().default("").catch(""),
  pageSize: z.coerce
    .number()
    .pipe(z.union([z.literal(20), z.literal(50), z.literal(100)]))
    .catch(20),
});

// Migration shim: this flat route accepts both admin and participant grade
// permissions until WU-7 (Slice 3) splits it into /admin/grades and /app/grades.
// Delete this file in WU-8 once the new area routes are live.
export const Route = createFileRoute("/_authenticated/grades")({
  validateSearch: gradesSearchSchema,
  beforeLoad: ({ context }) => {
    requireAnyPermission(context.session, [
      "grades.read",
      "grades.write",
      "grades.view_own",
    ]);
  },
  component: GradesPage,
});
