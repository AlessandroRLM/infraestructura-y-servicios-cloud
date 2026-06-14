import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";
import { AcademicsPage } from "@/features/academics";

const academicsSearchSchema = z.object({
  tab: z.enum(["programs", "courses"]).default("programs").catch("programs"),
});

export const Route = createFileRoute("/_authenticated/academics")({
  validateSearch: academicsSearchSchema,
  component: AcademicsPage,
});
