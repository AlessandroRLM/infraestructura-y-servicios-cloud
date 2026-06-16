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
import { useCourses } from "@/features/academics";

interface CourseSchemePicker {
  /** Currently selected course id, or empty string when nothing is selected. */
  value: string;
  /** Called with the selected course id when the user picks a course. */
  onChange: (courseId: string) => void;
}

/**
 * Searchable course combobox backed by the catalog ListCourses RPC via useCourses.
 * Uses a Popover + Command pattern, matching ProgramCoursesManager.
 * The search query is managed locally and passed to useCourses for server-side filtering.
 */
export function CourseSchemePicker({ value, onChange }: CourseSchemePicker) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");

  const { courses, isLoading } = useCourses(query);

  const selectedCourse = courses.find((c) => c.id === value);
  const label = selectedCourse
    ? `${selectedCourse.code} — ${selectedCourse.name}`
    : "Seleccionar asignatura";

  const handleSelect = (courseId: string) => {
    onChange(courseId);
    setOpen(false);
    setQuery("");
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-label="Seleccionar asignatura"
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
