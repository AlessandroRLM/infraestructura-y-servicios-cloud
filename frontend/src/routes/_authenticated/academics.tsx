import { createFileRoute } from "@tanstack/react-router";
import { AcademicsPage } from "@/features/academics";

export const Route = createFileRoute("/_authenticated/academics")({
  component: AcademicsPage,
});
