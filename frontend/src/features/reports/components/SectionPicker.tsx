import { ChevronsUpDown } from "lucide-react";
import { useEffect, useState } from "react";
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
import { academicPeriodLabel, useOwnSections } from "@/core/catalog";
import type { TeachingSection } from "@/gen/catalog/v1/catalog_pb";

/**
 * Returns an enriched display label for a TeachingSection.
 * Format: "MAT101 · Cálculo I — 2026 · Semestre 1"
 * Falls back gracefully when course or period fields are empty.
 */
export function buildSectionLabel(section: TeachingSection): string {
  const { courseCode, courseName, periodYear, periodTerm } = section;

  if (courseCode && courseName && periodYear && periodTerm) {
    return `${courseCode} · ${courseName} — ${academicPeriodLabel(periodYear, periodTerm)}`;
  }
  if (courseCode && courseName) {
    return `${courseCode} · ${courseName}`;
  }
  // Graceful fallback: short ID
  return `Sección ${section.id.slice(0, 8)}…`;
}

interface SectionPickerProps {
  /** Currently selected section id, or empty string when nothing is selected. */
  value: string;
  /** Called with the selected section id when the user picks a section. */
  onChange: (sectionId: string) => void;
}

/**
 * Searchable section combobox backed by ListOwnSections (role-aware RPC) via useOwnSections.
 * Teachers receive their own sections; admins receive all sections — no client-side
 * permission branching needed, the backend discriminates by role.
 *
 * Labels are built directly from TeachingSection.courseCode / courseName / periodYear /
 * periodTerm — no secondary course or period lookups required.
 *
 * The selected label is stored at pick-time (selectedLabel) so it persists
 * independently of the current query page — prevents label vanishing after paginating.
 * Mirrors CourseSchemePicker (Popover + Command pattern).
 */
export function SectionPicker({ value, onChange }: SectionPickerProps) {
  const [open, setOpen] = useState(false);
  const [selectedLabel, setSelectedLabel] = useState<string | null>(null);

  const {
    sections,
    isLoading: sectionsLoading,
    hasNextPage,
    isFetchingNextPage,
    fetchNextPage,
  } = useOwnSections();

  // Drain all pages while the popover is open so client-side search covers every section.
  // Gated on `open` to avoid eager background network traffic before the user interacts.
  useEffect(() => {
    if (open && hasNextPage && !isFetchingNextPage) fetchNextPage();
  }, [open, hasNextPage, isFetchingNextPage, fetchNextPage]);

  const isLoading = sectionsLoading;

  const liveSection = sections.find((s) => s.id === value);

  const label =
    value === ""
      ? "Seleccionar sección"
      : (selectedLabel ??
        (liveSection ? buildSectionLabel(liveSection) : "Seleccionar sección"));

  const ariaLabel =
    value === ""
      ? "Seleccionar sección"
      : (selectedLabel ?? "Seleccionar sección");

  const handleSelect = (sectionId: string) => {
    const picked = sections.find((s) => s.id === sectionId);
    setSelectedLabel(picked ? buildSectionLabel(picked) : null);
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
                      keywords={[buildSectionLabel(s)]}
                      onSelect={() => handleSelect(s.id)}
                    >
                      {buildSectionLabel(s)}
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
