import { ChevronsUpDown } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  academicPeriodLabel,
  useAcademicPeriods,
  useCourses,
  useSections,
} from "@/core/catalog";
import type {
  AcademicPeriod,
  Course,
  Section,
} from "@/gen/catalog/v1/catalog_pb";

/**
 * Returns an enriched display label for a section, joining course + period data.
 * Format: "MAT101 · Cálculo I — 2026 · Semestre 1"
 * Falls back gracefully if course or period data is missing.
 */
export function buildSectionLabel(
  section: Section,
  courseMap: Map<string, Course>,
  periodMap: Map<string, AcademicPeriod>,
): string {
  const course = courseMap.get(section.courseId);
  const period = periodMap.get(section.academicPeriodId);

  if (course && period) {
    return `${course.code} · ${course.name} — ${academicPeriodLabel(period.year, period.term)}`;
  }
  if (course) {
    return `${course.code} · ${course.name}`;
  }
  // Graceful fallback: short ID (original behaviour)
  return `Sección ${section.id.slice(0, 8)}…`;
}

interface SectionPickerProps {
  /** Currently selected section id, or empty string when nothing is selected. */
  value: string;
  /** Called with the selected section id when the user picks a section. */
  onChange: (sectionId: string) => void;
}

/**
 * Searchable section combobox backed by the catalog ListSections RPC via useSections.
 * Enriches labels with course code + name + period (year · Semestre N) via joint lookups.
 * Mirrors CourseSchemePicker (Popover + Command pattern).
 *
 * The selected label is stored at pick-time (selectedLabel) so it persists
 * independently of the current query page — prevents label vanishing after paginating.
 */
export function SectionPicker({ value, onChange }: SectionPickerProps) {
  const [open, setOpen] = useState(false);
  const [selectedLabel, setSelectedLabel] = useState<string | null>(null);

  const { sections, isLoading: sectionsLoading } = useSections();
  const { courses } = useCourses();
  const { periods } = useAcademicPeriods();

  const isLoading = sectionsLoading;

  // Build lookup maps for O(1) join — Compiler handles memoization.
  const courseMap = new Map<string, Course>(courses.map((c) => [c.id, c]));
  const periodMap = new Map<string, AcademicPeriod>(
    periods.map((p) => [p.id, p]),
  );

  const liveSection = sections.find((s) => s.id === value);

  const label =
    value === ""
      ? "Seleccionar sección"
      : (selectedLabel ??
        (liveSection
          ? buildSectionLabel(liveSection, courseMap, periodMap)
          : "Seleccionar sección"));

  const ariaLabel =
    value === ""
      ? "Seleccionar sección"
      : (selectedLabel ?? "Seleccionar sección");

  const handleSelect = (sectionId: string) => {
    const picked = sections.find((s) => s.id === sectionId);
    setSelectedLabel(
      picked ? buildSectionLabel(picked, courseMap, periodMap) : null,
    );
    onChange(sectionId);
    setOpen(false);
  };

  const handleOpenChange = (nextOpen: boolean) => {
    setOpen(nextOpen);
  };

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-label={ariaLabel}
          aria-expanded={open}
          aria-haspopup="listbox"
          className="w-full justify-between"
        >
          <span className="truncate">{label}</span>
          <ChevronsUpDown
            className="size-4 shrink-0 text-muted-foreground"
            aria-hidden
          />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="p-0 w-[var(--radix-popover-trigger-width)]">
        <Command>
          <CommandInput placeholder="Buscar sección…" />
          <CommandList>
            {isLoading ? (
              <div className="py-6 text-center text-sm text-muted-foreground">
                Cargando…
              </div>
            ) : (
              <>
                <CommandEmpty>No se encontraron secciones.</CommandEmpty>
                <CommandGroup>
                  {sections.map((s) => (
                    <CommandItem
                      key={s.id}
                      value={s.id}
                      onSelect={() => handleSelect(s.id)}
                    >
                      {buildSectionLabel(s, courseMap, periodMap)}
                    </CommandItem>
                  ))}
                </CommandGroup>
              </>
            )}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
