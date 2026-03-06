"use client";
import { useEffect, useState } from "react";
import { Search, X } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { cn } from "@/lib/utils";
import { useDebounce } from "@/hooks/use-debounce";

interface SearchInputProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  debounceMs?: number;
  loading?: boolean;
  className?: string;
}

export function SearchInput({
  value,
  onChange,
  placeholder = "Search...",
  debounceMs = 300,
  loading = false,
  className,
}: SearchInputProps) {
  const [internal, setInternal] = useState(value);
  const debounced = useDebounce(internal, debounceMs);

  useEffect(() => {
    onChange(debounced);
  }, [debounced, onChange]);

  useEffect(() => {
    if (value !== internal) setInternal(value);
  }, [value]); // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div className={cn("relative", className)}>
      <div className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2">
        {loading ? (
          <Spinner className="h-4 w-4 text-muted-foreground" />
        ) : (
          <Search className="h-4 w-4 text-muted-foreground" aria-hidden />
        )}
      </div>
      <Input
        value={internal}
        onChange={(e) => setInternal(e.target.value)}
        placeholder={placeholder}
        className="pl-9 pr-9 h-8"
        onKeyDown={(e) => { if (e.key === "Escape") { setInternal(""); onChange(""); } }}
        aria-label={placeholder}
      />
      {internal && (
        <button
          className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-0.5 hover:bg-muted focus:outline-none focus:ring-1 focus:ring-ring"
          onClick={() => { setInternal(""); onChange(""); }}
          aria-label="Clear search"
          type="button"
        >
          <X className="h-4 w-4 text-muted-foreground" aria-hidden />
        </button>
      )}
    </div>
  );
}
