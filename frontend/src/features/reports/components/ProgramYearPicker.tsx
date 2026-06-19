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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { usePrograms } from "@/core/catalog";

interface ProgramYearPickerProps {
  /** Currently selected program id, or empty string when nothing is selected. */
  programId: string;
  /** Currently selected year (2000–2100), or undefined when not set. */
  year: number | undefined;
  /** Called when the user picks a program. */
  onProgramChange: (programId: string) => void;
  /** Called when the user changes the year field. */
  onYearChange: (year: number | undefined) => void;
}

/**
 * Combined program + year picker for the ProgramSummary report.
 * Both inputs must be set to enable the query (programId && year).
 *
 * - Program: Popover + Command combobox (mirrors SectionPicker).
 * - Year: plain number input (integer, 2000–2100).
 */
export function ProgramYearPicker({
  programId,
  year,
  onProgramChange,
  onYearChange,
}: ProgramYearPickerProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [selectedLabel, setSelectedLabel] = useState<string | null>(null);
  // Local text state so the input is not controlled by the URL-committed `year` prop.
  // This prevents the field from clearing mid-type (e.g. typing "2" resets "2026").
  const [yearText, setYearText] = useState(year != null ? String(year) : "");

  // Sync external prop changes (back/forward navigation) back into the local text.
  // Only sync when `year` is set: a transition to `undefined` is always driven by
  // in-progress local typing (partial/out-of-range), not navigation, so clearing the
  // field here would wipe what the user is typing.
  useEffect(() => {
    if (year != null) setYearText(String(year));
  }, [year]);

  const { programs, isLoading } = usePrograms(search);

  const liveProgram = programs.find((p) => p.id === programId);

  const programLabel =
    programId === ""
      ? "Seleccionar programa"
      : (selectedLabel ??
        (liveProgram
          ? `${liveProgram.code} — ${liveProgram.name}`
          : "Seleccionar programa"));

  const handleSelectProgram = (pid: string) => {
    const picked = programs.find((p) => p.id === pid);
    setSelectedLabel(picked ? `${picked.code} — ${picked.name}` : null);
    onProgramChange(pid);
    setSearch("");
    setOpen(false);
  };

  const handleYearChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const raw = e.target.value;
    // Always keep visible text in sync so the field does not clear mid-type.
    setYearText(raw);
    const trimmed = raw.trim();
    if (trimmed === "") {
      onYearChange(undefined);
      return;
    }
    const parsed = Number.parseInt(trimmed, 10);
    if (!Number.isNaN(parsed) && parsed >= 2000 && parsed <= 2100) {
      onYearChange(parsed);
    } else {
      // Partial / out-of-range input: commit undefined upstream but keep text visible.
      onYearChange(undefined);
    }
  };

  return (
    <div className="flex flex-col gap-3">
      <Popover
        open={open}
        onOpenChange={(o) => {
          setOpen(o);
          if (!o) setSearch("");
        }}
      >
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            role="combobox"
            aria-label={programLabel}
            aria-expanded={open}
            aria-haspopup="listbox"
            className="w-full justify-between"
          >
            <span className="truncate">{programLabel}</span>
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
                        value={`${p.code} — ${p.name}`}
                        onSelect={() => handleSelectProgram(p.id)}
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

      <div className="flex flex-col gap-1">
        <Label htmlFor="program-year-input">Año académico</Label>
        <Input
          id="program-year-input"
          type="number"
          min={2000}
          max={2100}
          placeholder="Ej. 2026"
          value={yearText}
          onChange={handleYearChange}
          className="max-w-[120px]"
        />
      </div>
    </div>
  );
}
