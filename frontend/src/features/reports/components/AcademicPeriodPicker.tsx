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
import { academicPeriodLabel, useAcademicPeriods } from "@/core/catalog";

interface AcademicPeriodPickerProps {
  /** Currently selected period id, or empty string when nothing is selected. */
  value: string;
  /** Called with the selected period id when the user picks a period. */
  onChange: (periodId: string) => void;
}

/**
 * Searchable academic period combobox backed by the catalog ListAcademicPeriods RPC.
 * Displays a readable label: "2026 · Semestre 1".
 * Mirrors SectionPicker (Popover + Command pattern).
 */
export function AcademicPeriodPicker({
  value,
  onChange,
}: AcademicPeriodPickerProps) {
  const [open, setOpen] = useState(false);
  const [selectedLabel, setSelectedLabel] = useState<string | null>(null);

  const { periods, isLoading } = useAcademicPeriods();

  const livePeriod = periods.find((p) => p.id === value);

  const label =
    value === ""
      ? "Seleccionar período"
      : (selectedLabel ??
        (livePeriod
          ? academicPeriodLabel(livePeriod.year, livePeriod.term)
          : "Seleccionar período"));

  const ariaLabel =
    value === ""
      ? "Seleccionar período"
      : (selectedLabel ?? "Seleccionar período");

  const handleSelect = (periodId: string) => {
    const picked = periods.find((p) => p.id === periodId);
    setSelectedLabel(
      picked ? academicPeriodLabel(picked.year, picked.term) : null,
    );
    onChange(periodId);
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
          <CommandInput placeholder="Buscar período…" />
          <CommandList>
            {isLoading ? (
              <div className="py-6 text-center text-sm text-muted-foreground">
                Cargando…
              </div>
            ) : (
              <>
                <CommandEmpty>No se encontraron períodos.</CommandEmpty>
                <CommandGroup>
                  {periods.map((p) => (
                    <CommandItem
                      key={p.id}
                      value={p.id}
                      onSelect={() => handleSelect(p.id)}
                    >
                      {academicPeriodLabel(p.year, p.term)}
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
