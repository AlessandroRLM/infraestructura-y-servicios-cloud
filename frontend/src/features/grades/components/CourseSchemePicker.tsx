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
import { useCourses } from "@/core/catalog";
import { useDebounce } from "@/core/hooks";

/** Milliseconds to debounce the course search query before issuing a ListCourses RPC. */
const SEARCH_DEBOUNCE_MS = 300;

interface CourseSchemePickerProps {
  /** Currently selected course id, or empty string when nothing is selected. */
  value: string;
  /** Called with the selected course id when the user picks a course. */
  onChange: (courseId: string) => void;
}

/**
 * Searchable course combobox backed by the catalog ListCourses RPC via useCourses.
 * Uses a Popover + Command pattern, matching ProgramCoursesManager.
 * The search query is managed locally and passed to useCourses for server-side filtering.
 * The input is controlled (immediate visual update); the RPC query is debounced
 * via useDebounce to avoid one ListCourses request per keystroke.
 *
 * The selected course label is stored at pick-time (selectedLabel) so it persists
 * independently of the current query page — prevents label vanishing after a search.
 */
export function CourseSchemePicker({
  value,
  onChange,
}: CourseSchemePickerProps) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  // Snapshot the display label at pick-time, independent of the current page.
  const [selectedLabel, setSelectedLabel] = useState<string | null>(null);

  const debouncedQuery = useDebounce(query, SEARCH_DEBOUNCE_MS);
  const { courses, isLoading } = useCourses(debouncedQuery);

  // Prefer the snapshotted label; fall back to live lookup for the initial render
  // when value is pre-set and selectedLabel has not been set yet.
  const liveCourse = courses.find((c) => c.id === value);
  const label =
    value === ""
      ? "Seleccionar asignatura"
      : (selectedLabel ??
        (liveCourse
          ? `${liveCourse.code} — ${liveCourse.name}`
          : "Seleccionar asignatura"));

  const ariaLabel =
    value === ""
      ? "Seleccionar asignatura"
      : (selectedLabel ?? "Seleccionar asignatura");

  const handleSelect = (courseId: string) => {
    const picked = courses.find((c) => c.id === courseId);
    setSelectedLabel(picked ? `${picked.code} — ${picked.name}` : null);
    onChange(courseId);
    setOpen(false);
    setQuery("");
  };

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) {
      // Clear the search query when popover closes without selection
      setQuery("");
    }
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
        <Command shouldFilter={false}>
          <CommandInput
            placeholder="Buscar asignatura…"
            value={query}
            onValueChange={setQuery}
          />
          <CommandList>
            {isLoading ? (
              <div className="py-6 text-center text-sm text-muted-foreground">
                Cargando…
              </div>
            ) : (
              <>
                <CommandEmpty>No se encontraron asignaturas.</CommandEmpty>
                <CommandGroup>
                  {courses.map((c) => (
                    <CommandItem
                      key={c.id}
                      value={c.id}
                      onSelect={() => handleSelect(c.id)}
                    >
                      {c.code} — {c.name}
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
