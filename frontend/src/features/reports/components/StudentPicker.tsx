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
import { useUsers } from "@/core/iam";
import type { UserSummary } from "@/gen/iam/v1/iam_pb";

/**
 * Returns a readable display label for a user.
 * Format: "Given Names Paternal — email"
 * Falls back to email alone when display_name is absent or equals the email.
 */
export function buildUserLabel(user: UserSummary): string {
  const name = user.displayName.trim();
  if (!name || name === user.email) return user.email;
  return `${name} — ${user.email}`;
}

interface StudentPickerProps {
  /** Currently selected student id, or empty string when nothing is selected. */
  value: string;
  /** Called with the selected student id when the user picks a student. */
  onChange: (studentId: string) => void;
}

/**
 * Searchable student combobox backed by IamService.listUsers (core/iam/useUsers).
 * Filters by typing email or display_name — the RPC supports free-text search.
 * Mirrors AcademicPeriodPicker (Popover + Command pattern).
 *
 * NOTE: uses core/iam/useUsers to avoid a deep cross-feature import from
 * features/users (which only exports UsersPage from its barrel).
 */
export function StudentPicker({ value, onChange }: StudentPickerProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [selectedLabel, setSelectedLabel] = useState<string | null>(null);

  const { users, isLoading } = useUsers(search, 50);

  const liveUser = users.find((u) => u.id === value);

  const label =
    value === ""
      ? "Seleccionar estudiante"
      : (selectedLabel ??
        (liveUser ? buildUserLabel(liveUser) : "Seleccionar estudiante"));

  const ariaLabel =
    value === ""
      ? "Seleccionar estudiante"
      : (selectedLabel ?? "Seleccionar estudiante");

  const handleSelect = (userId: string) => {
    const picked = users.find((u) => u.id === userId);
    setSelectedLabel(picked ? buildUserLabel(picked) : null);
    onChange(userId);
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
            placeholder="Buscar por nombre o email…"
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
                <CommandEmpty>No se encontraron usuarios.</CommandEmpty>
                <CommandGroup>
                  {users.map((u) => (
                    <CommandItem
                      key={u.id}
                      value={u.id}
                      onSelect={() => handleSelect(u.id)}
                    >
                      {buildUserLabel(u)}
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
