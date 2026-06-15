import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";
import { requireRoutePermission } from "@/features/auth";
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

// Admin grades route — renders the admin/teacher GradesPage.
// The feature guard accepts grades.read or grades.write (teacher overlap).
// See ROUTE_PERMISSIONS["/admin/grades"].
export const Route = createFileRoute("/_authenticated/admin/grades")({
  validateSearch: gradesSearchSchema,
  beforeLoad: ({ context }) => {
    requireRoutePermission(context.session, "/admin/grades");
  },
  component: GradesPage,
});
