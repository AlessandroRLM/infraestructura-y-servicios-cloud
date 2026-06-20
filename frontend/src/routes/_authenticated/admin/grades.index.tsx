import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";
import { GradesPage } from "@/features/grades";

const gradesSearchSchema = z.object({
  q: z.string().default("").catch(""),
  pageSize: z.coerce
    .number()
    .pipe(z.union([z.literal(20), z.literal(50), z.literal(100)]))
    .default(50)
    .catch(50),
});

function GradesIndexPage() {
  const { q, pageSize } = Route.useSearch();
  const navigate = Route.useNavigate();

  return (
    <GradesPage
      q={q}
      pageSize={pageSize}
      onQueryChange={(v) => navigate({ search: (prev) => ({ ...prev, q: v }) })}
      onPageSizeChange={(n) =>
        navigate({ search: (prev) => ({ ...prev, pageSize: n }) })
      }
    />
  );
}

// Section selection table rendered at exactly /admin/grades.
// The parent grades.tsx layout applies the route permission guard; this index
// route just renders the page component.
export const Route = createFileRoute("/_authenticated/admin/grades/")({
  validateSearch: gradesSearchSchema,
  component: GradesIndexPage,
});
