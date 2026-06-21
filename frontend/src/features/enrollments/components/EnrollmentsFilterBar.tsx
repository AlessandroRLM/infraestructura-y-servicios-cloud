import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { PageSizeSelector, SearchInput } from "@/core/components";
import { SEARCH_DEBOUNCE_MS } from "../constants";
import type { AdminEnrollmentsSearch } from "../schemas/search";

/** Sentinel used in the Radix Select to represent "no filter" (empty string
 *  is forbidden by Radix as an item value). Converted to `undefined` before
 *  it reaches the caller. */
const ALL_STATUS = "__all__";

interface EnrollmentsFilterBarProps {
  q: string;
  year: number | undefined;
  status: AdminEnrollmentsSearch["status"];
  pageSize: number;
  onQChange: (q: string) => void;
  onYearChange: (year: number | undefined) => void;
  onStatusChange: (status: AdminEnrollmentsSearch["status"]) => void;
  onPageSizeChange: (n: number) => void;
}

/**
 * Presentational filter bar for the admin enrollments table.
 * All URL navigation lives in the consuming component (EnrollmentsTable).
 */
export function EnrollmentsFilterBar({
  q,
  year,
  status,
  pageSize,
  onQChange,
  onYearChange,
  onStatusChange,
  onPageSizeChange,
}: EnrollmentsFilterBarProps) {
  // Local year text buffer (untypeable-year fix): allows the user to type
  // digit-by-digit without the field being reset on every keystroke.
  const handleYearChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const raw = e.target.value;
    const trimmed = raw.trim();
    if (trimmed === "") {
      onYearChange(undefined);
      return;
    }
    const parsed = Number.parseInt(trimmed, 10);
    if (!Number.isNaN(parsed) && parsed >= 2000 && parsed <= 2100) {
      onYearChange(parsed);
    } else {
      onYearChange(undefined);
    }
  };

  const handleStatusChange = (value: string) => {
    if (value === ALL_STATUS) {
      onStatusChange(undefined);
    } else {
      onStatusChange(value as AdminEnrollmentsSearch["status"]);
    }
  };

  return (
    <div className="flex flex-wrap items-end gap-3">
      <SearchInput
        value={q}
        onChange={onQChange}
        debounceMs={SEARCH_DEBOUNCE_MS}
        placeholder="Buscar por estudiante o programa…"
        className="max-w-sm"
      />

      <div className="flex flex-col gap-1">
        <Label htmlFor="filter-year" className="text-xs">
          Año
        </Label>
        <Input
          id="filter-year"
          type="number"
          min={2000}
          max={2100}
          placeholder="Año"
          defaultValue={year ?? ""}
          onChange={handleYearChange}
          className="w-[100px]"
        />
      </div>

      <div className="flex flex-col gap-1">
        <Label htmlFor="filter-status" className="sr-only">
          Estado
        </Label>
        <Select value={status ?? ALL_STATUS} onValueChange={handleStatusChange}>
          <SelectTrigger
            id="filter-status"
            aria-label="Estado"
            className="w-[160px]"
          >
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL_STATUS}>Todas</SelectItem>
            <SelectItem value="pending">Pendiente</SelectItem>
            <SelectItem value="paid">Pagada</SelectItem>
            <SelectItem value="cancelled">Cancelada</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <PageSizeSelector value={pageSize} onChange={onPageSizeChange} />
    </div>
  );
}
