import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

interface PageSizeSelectorProps {
  /** Currently selected page size. */
  value: number;
  /** Called with the new page size when the user changes the selection. */
  onChange: (n: number) => void;
  /** Available page size options (default: [20, 50, 100]). */
  options?: number[];
}

/**
 * Controlled page-size selector built on the shadcn Select primitive.
 * URL-sync is the caller's responsibility; this component owns no router logic.
 */
export function PageSizeSelector({
  value,
  onChange,
  options = [20, 50, 100],
}: PageSizeSelectorProps) {
  return (
    <Select value={String(value)} onValueChange={(v) => onChange(Number(v))}>
      <SelectTrigger aria-label="Filas por página">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {options.map((n) => (
          <SelectItem key={n} value={String(n)}>
            {n} por página
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
