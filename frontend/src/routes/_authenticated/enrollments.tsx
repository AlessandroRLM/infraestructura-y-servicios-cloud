import { createFileRoute } from "@tanstack/react-router";
import { EnrollmentsPage } from "@/features/enrollments";

export const Route = createFileRoute("/_authenticated/enrollments")({
  component: EnrollmentsPage,
});
