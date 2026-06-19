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
  const [selectedLabel, setSelectedLabel] = useState<string | null>(null);

  const { programs, isLoading } = usePrograms();

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
    setOpen(false);
  };

  const handleYearChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const raw = e.target.value.trim();
    if (raw === "") {
      onYearChange(undefined);
      return;
    }
    const parsed = Number.parseInt(raw, 10);
    if (!Number.isNaN(parsed) && parsed >= 2000 && parsed <= 2100) {
      onYearChange(parsed);
    } else {
      onYearChange(undefined);
    }
  };

  return (
    <div className="flex flex-col gap-3">
      <Popover open={open} onOpenChange={setOpen}>
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
          <Command>
            <CommandInput placeholder="Buscar programa…" />
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
          defaultValue={year ?? ""}
          onChange={handleYearChange}
          className="max-w-[120px]"
        />
      </div>
    </div>
  );
}
