import { createFileRoute } from "@tanstack/react-router";
import { GradesPage } from "@/features/grades";

// Section selection table rendered at exactly /admin/grades.
// The parent grades.tsx layout applies the route permission guard; this index
// route just renders the page component.
export const Route = createFileRoute("/_authenticated/admin/grades/")({
  component: GradesPage,
});
