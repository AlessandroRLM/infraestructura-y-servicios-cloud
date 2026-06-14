import { useEffect, useRef, useState } from "react";
import { Input } from "@/components/ui/input";
import { useDebounce } from "@/core/hooks";

interface SearchInputProps {
  /** Controlled value — seed from the URL search param. */
  value: string;
  /** Called with the debounced value when it changes. */
  onChange: (value: string) => void;
  /** Debounce window in ms (default: 300). */
  debounceMs?: number;
  placeholder?: string;
  className?: string;
}

/**
 * Controlled search input with built-in debounce.
 * URL-sync stays in the consuming component; this component only owns the
 * visual input state and the debounce timer.
 */
export function SearchInput({
  value,
  onChange,
  debounceMs = 300,
  placeholder,
  className,
}: SearchInputProps) {
  const [inputValue, setInputValue] = useState(value);
  const debounced = useDebounce(inputValue, debounceMs);
  // Tracks the last value seen from the parent or committed via onChange, so the
  // debounce effect distinguishes a real edit from an external reset without
  // depending on `value` (which would re-run it spuriously / read it stale).
  const committedRef = useRef(value);

  // Sync external value changes (e.g. tab switch resets q to "") into local state.
  useEffect(() => {
    committedRef.current = value;
    setInputValue(value);
  }, [value]);

  // Fire onChange only when the debounced input diverges from the committed value.
  useEffect(() => {
    if (debounced !== committedRef.current) {
      committedRef.current = debounced;
      onChange(debounced);
    }
  }, [debounced, onChange]);

  return (
    <Input
      placeholder={placeholder}
      value={inputValue}
      onChange={(e) => setInputValue(e.target.value)}
      className={className}
    />
  );
}
