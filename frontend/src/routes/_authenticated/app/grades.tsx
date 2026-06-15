import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";
import { requireRoutePermission } from "@/features/auth";
import { OwnGradesView } from "@/features/grades";

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

/**
 * Route-owned wrapper that reads search params and passes them down as props.
 * Keeps OwnGradesView free of any direct Route import, breaking the circular
 * dependency that would arise from app/grades.tsx ↔ OwnGradesView.
 */
function AppGradesPage() {
  const { period, program, pageSize } = Route.useSearch();
  const navigate = Route.useNavigate();

  return (
    <OwnGradesView
      period={period}
      program={program}
      pageSize={pageSize}
      onPeriodChange={(v) =>
        navigate({ search: (prev) => ({ ...prev, period: v }) })
      }
      onProgramChange={(v) =>
        navigate({ search: (prev) => ({ ...prev, program: v }) })
      }
    />
  );
}

// Participant grades route — renders the student's own-grades view directly.
// The feature guard requires grades.view_own. See ROUTE_PERMISSIONS["/app/grades"].
export const Route = createFileRoute("/_authenticated/app/grades")({
  validateSearch: gradesSearchSchema,
  beforeLoad: ({ context }) => {
    requireRoutePermission(context.session, "/app/grades");
  },
  component: AppGradesPage,
});
