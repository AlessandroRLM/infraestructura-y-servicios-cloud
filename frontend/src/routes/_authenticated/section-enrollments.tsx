import { createFileRoute } from "@tanstack/react-router";
import { SectionEnrollmentsPage } from "@/features/section-enrollments";

export const Route = createFileRoute("/_authenticated/section-enrollments")({
  component: SectionEnrollmentsPage,
});
