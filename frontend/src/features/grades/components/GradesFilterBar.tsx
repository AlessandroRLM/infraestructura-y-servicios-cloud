import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { ProgramOption } from "../hooks/useOwnEnrollmentsForFilter";
import type { PeriodOption } from "../hooks/useOwnGradePeriods";

interface GradesFilterBarProps {
  periods: PeriodOption[];
  programs: ProgramOption[];
  isLoadingPeriods: boolean;
  /** Current academic-period filter value; empty string means no filter. */
  period: string;
  /** Current program filter value; empty string means no filter. */
  program: string;
  onPeriodChange: (periodId: string) => void;
  onProgramChange: (programId: string) => void;
}

/** Sentinel value representing "no filter applied" in the Select. */
const ALL_VALUE = "__all__";

/**
 * Filter bar for the "Mis notas" view. Renders two Select controls:
 * - Período: academic period filter
 * - Carrera: program (carrera) filter
 * Both controls receive their current value and change handlers from the parent;
 * URL-state management lives in OwnGradesView.
 */
export function GradesFilterBar({
  periods,
  programs,
  isLoadingPeriods,
  period,
  program,
  onPeriodChange,
  onProgramChange,
}: GradesFilterBarProps) {
  const handlePeriodChange = (value: string) => {
    onPeriodChange(value === ALL_VALUE ? "" : value);
  };

  const handleProgramChange = (value: string) => {
    onProgramChange(value === ALL_VALUE ? "" : value);
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
