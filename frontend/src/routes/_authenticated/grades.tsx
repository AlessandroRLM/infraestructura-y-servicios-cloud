import { createFileRoute } from "@tanstack/react-router";
import { GradesPage } from "@/features/grades";

export const Route = createFileRoute("/_authenticated/grades")({
  component: GradesPage,
});
