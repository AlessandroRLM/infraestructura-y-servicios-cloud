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
import { usePrograms } from "@/core/catalog";

interface ProgramPickerProps {
  /** Currently selected program id, or empty string when nothing is selected. */
  value: string;
  /** Called with the selected program id when the user picks a program. */
  onChange: (programId: string) => void;
}

/**
 * Feature-local searchable program combobox (Popover + Command pattern).
 * Mirrors StudentPicker shape: free-text search, caches selectedLabel at pick time.
 * Backed by core/catalog/usePrograms (CatalogService.listPrograms).
 * Used only by CreateEnrollmentDialog.
 */
export function ProgramPicker({ value, onChange }: ProgramPickerProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [selectedLabel, setSelectedLabel] = useState<string | null>(null);

  const { programs, isLoading } = usePrograms(search);

  const liveProgram = programs.find((p) => p.id === value);

  const label =
    value === ""
      ? "Seleccionar programa"
      : (selectedLabel ??
        (liveProgram
          ? `${liveProgram.code} — ${liveProgram.name}`
          : "Seleccionar programa"));

  const ariaLabel =
    value === ""
      ? "Seleccionar programa"
      : (selectedLabel ?? "Seleccionar programa");

  const handleSelect = (programId: string) => {
    const picked = programs.find((p) => p.id === programId);
    setSelectedLabel(picked ? `${picked.code} — ${picked.name}` : null);
    onChange(programId);
    setSearch(""); // clear search on select (mirrors ProgramYearPicker)
    setOpen(false);
  };

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) setSearch("");
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
            placeholder="Buscar programa…"
            value={search}
            onValueChange={setSearch}
          />
          <CommandList>
            {isLoading ? (
              <div className="py-6 text-center text-sm text-muted-foreground">
                Cargando…
              </div>
            ) : (
              <>
                <CommandEmpty>No se encontraron programas.</CommandEmpty>
                <CommandGroup>
                  {programs.map((p) => (
                    <CommandItem
                      key={p.id}
                      value={p.id}
                      onSelect={() => handleSelect(p.id)}
                    >
                      {p.code} — {p.name}
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
