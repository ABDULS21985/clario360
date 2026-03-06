"use client";
import { useState, useRef } from "react";
import { X, ChevronDown, Check } from "lucide-react";
import { cn } from "@/lib/utils";

interface MultiSelectProps {
  options: Array<{ label: string; value: string }>;
  selected: string[];
  onChange: (values: string[]) => void;
  placeholder?: string;
  searchable?: boolean;
  maxSelected?: number;
  disabled?: boolean;
  className?: string;
}

export function MultiSelect({
  options,
  selected,
  onChange,
  placeholder = "Select...",
  searchable = true,
  maxSelected,
  disabled = false,
  className,
}: MultiSelectProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  const filteredOptions = options.filter((o) =>
    o.label.toLowerCase().includes(search.toLowerCase())
  );

  const toggle = (value: string) => {
    if (selected.includes(value)) {
      onChange(selected.filter((v) => v !== value));
    } else if (!maxSelected || selected.length < maxSelected) {
      onChange([...selected, value]);
    }
  };

  const remove = (value: string, e: React.MouseEvent) => {
    e.stopPropagation();
    onChange(selected.filter((v) => v !== value));
  };

  return (
    <div className={cn("relative", className)}>
      <div
        className={cn(
          "flex min-h-9 flex-wrap gap-1 rounded-md border border-input bg-background px-2 py-1 cursor-pointer",
          open && "ring-2 ring-ring",
          disabled && "opacity-50 cursor-not-allowed"
        )}
        onClick={() => { if (!disabled) { setOpen(!open); inputRef.current?.focus(); } }}
        role="combobox"
        aria-expanded={open}
        aria-haspopup="listbox"
      >
        {selected.map((v) => {
          const opt = options.find((o) => o.value === v);
          return (
            <span key={v} className="inline-flex items-center gap-1 rounded bg-secondary px-1.5 py-0.5 text-xs font-medium">
              {opt?.label ?? v}
              <button type="button" onClick={(e) => remove(v, e)} aria-label={`Remove ${opt?.label ?? v}`} className="hover:text-destructive focus:outline-none">
                <X className="h-3 w-3" aria-hidden />
              </button>
            </span>
          );
        })}
        {searchable && open ? (
          <input
            ref={inputRef}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="flex-1 min-w-[60px] bg-transparent text-sm outline-none placeholder:text-muted-foreground"
            placeholder={selected.length === 0 ? placeholder : ""}
            onBlur={() => setTimeout(() => { setOpen(false); setSearch(""); }, 150)}
          />
        ) : selected.length === 0 ? (
          <span className="text-sm text-muted-foreground flex-1 py-0.5">{placeholder}</span>
        ) : null}
        <ChevronDown className="ml-auto h-4 w-4 text-muted-foreground shrink-0 self-center" aria-hidden />
      </div>

      {open && (
        <div className="absolute z-50 mt-1 w-full rounded-md border border-border bg-popover shadow-md">
          <div className="max-h-60 overflow-y-auto p-1" role="listbox" aria-multiselectable="true">
            {filteredOptions.length === 0 ? (
              <div className="px-2 py-4 text-sm text-muted-foreground text-center">No options found.</div>
            ) : (
              filteredOptions.map((option) => (
                <button
                  key={option.value}
                  role="option"
                  aria-selected={selected.includes(option.value)}
                  className="flex items-center gap-2 w-full rounded px-2 py-1.5 text-sm hover:bg-muted focus:outline-none focus:bg-muted"
                  onClick={(e) => { e.preventDefault(); toggle(option.value); }}
                  type="button"
                >
                  <div className={cn(
                    "flex h-4 w-4 items-center justify-center rounded border border-primary",
                    selected.includes(option.value) ? "bg-primary text-primary-foreground" : "opacity-30"
                  )}>
                    {selected.includes(option.value) && <Check className="h-3 w-3" />}
                  </div>
                  {option.label}
                </button>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  );
}
