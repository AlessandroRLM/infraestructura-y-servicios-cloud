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

export const Route = createFileRoute("/_authenticated/grades")({
  validateSearch: gradesSearchSchema,
  beforeLoad: ({ context }) => {
    requireRoutePermission(context.session, "/grades");
  },
  component: GradesPage,
});
