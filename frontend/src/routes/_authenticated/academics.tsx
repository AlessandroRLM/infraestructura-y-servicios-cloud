import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";
import { AcademicsPage } from "@/features/academics";
import { requireRoutePermission } from "@/features/auth";

const academicsSearchSchema = z.object({
  tab: z.enum(["programs", "courses"]).default("programs").catch("programs"),
});

export const Route = createFileRoute("/_authenticated/academics")({
  validateSearch: academicsSearchSchema,
  beforeLoad: ({ context }) => {
    requireRoutePermission(context.session, "/academics");
  },
  component: AcademicsPage,
});
