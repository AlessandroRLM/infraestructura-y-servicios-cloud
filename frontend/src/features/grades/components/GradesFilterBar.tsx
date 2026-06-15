import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Route } from "@/routes/_authenticated/grades";
import type { ProgramOption } from "../hooks/useOwnEnrollmentsForFilter";
import type { PeriodOption } from "../hooks/useOwnGradePeriods";

interface GradesFilterBarProps {
  periods: PeriodOption[];
  programs: ProgramOption[];
  isLoadingPeriods: boolean;
}

/** Sentinel value representing "no filter applied" in the Select. */
const ALL_VALUE = "__all__";

/**
 * Filter bar for the "Mis notas" view. Renders two Select controls:
 * - Período: academic period filter
 * - Carrera: program (carrera) filter
 * Both controls read their current value from the URL and navigate to update it,
 * following the academics q/pageSize URL-state pattern.
 */
export function GradesFilterBar({
  periods,
  programs,
  isLoadingPeriods,
}: GradesFilterBarProps) {
  const { period, program } = Route.useSearch();
  const navigate = Route.useNavigate();

  const handlePeriodChange = (value: string) => {
    navigate({
      search: (prev) => ({
        ...prev,
        period: value === ALL_VALUE ? "" : value,
      }),
    });
  };

  const handleProgramChange = (value: string) => {
    navigate({
      search: (prev) => ({
        ...prev,
        program: value === ALL_VALUE ? "" : value,
      }),
    });
  };

  return (
    <div className="flex flex-wrap items-center gap-3">
      <Select
        value={period || ALL_VALUE}
        onValueChange={handlePeriodChange}
        disabled={isLoadingPeriods}
      >
        <SelectTrigger className="w-40" aria-label="Filtrar por período">
          <SelectValue placeholder="Período" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={ALL_VALUE}>Todos los períodos</SelectItem>
          {periods.map((p) => (
            <SelectItem key={p.id} value={p.id}>
              {p.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <Select value={program || ALL_VALUE} onValueChange={handleProgramChange}>
        <SelectTrigger className="w-52" aria-label="Filtrar por carrera">
          <SelectValue placeholder="Carrera" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={ALL_VALUE}>Todas las carreras</SelectItem>
          {programs.map((p) => (
            <SelectItem key={p.id} value={p.id}>
              {p.name || p.id}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
