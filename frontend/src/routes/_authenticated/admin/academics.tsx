import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";
import { AcademicsPage } from "@/features/academics";
import { requireRoutePermission } from "@/features/auth";

const academicsSearchSchema = z.object({
  tab: z
    .enum(["programs", "courses", "sections", "periods"])
    .default("programs")
    .catch("programs"),
  q: z.string().default("").catch(""),
  pageSize: z.coerce
    .number()
    .pipe(z.union([z.literal(20), z.literal(50), z.literal(100)]))
    .catch(20),
});

export const Route = createFileRoute("/_authenticated/admin/academics")({
  validateSearch: academicsSearchSchema,
  beforeLoad: ({ context }) => {
    requireRoutePermission(context.session, "/admin/academics");
  },
  component: AcademicsPage,
});
