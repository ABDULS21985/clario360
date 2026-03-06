"use client";
import { useState } from "react";
import { Check, ChevronDown } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import type { FilterConfig } from "@/types/table";

interface DataTableFilterProps {
  config: FilterConfig;
  value: string | string[] | undefined;
  onChange: (key: string, value: string | string[] | undefined) => void;
}

export function DataTableFilter({
  config,
  value,
  onChange,
}: DataTableFilterProps) {
  const [open, setOpen] = useState(false);

  const isActive =
    value !== undefined &&
    value !== "" &&
    !(Array.isArray(value) && value.length === 0);
  const selectedCount = Array.isArray(value)
    ? value.length
    : isActive
      ? 1
      : 0;

  if (config.type === "select" || config.type === "multi-select") {
    const isMulti = config.type === "multi-select";
    const selected = Array.isArray(value) ? value : value ? [value] : [];

    const toggleOption = (optValue: string) => {
      if (isMulti) {
        const next = selected.includes(optValue)
          ? selected.filter((v) => v !== optValue)
          : [...selected, optValue];
        onChange(config.key, next.length > 0 ? next : undefined);
      } else {
        onChange(
          config.key,
          selected[0] === optValue ? undefined : optValue
        );
        setOpen(false);
      }
    };

    return (
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            size="sm"
            className={cn("h-8 border-dashed", isActive && "border-primary")}
          >
            {config.label}
            {selectedCount > 0 && (
              <span className="ml-2 rounded-full bg-primary text-primary-foreground text-xs px-1.5 py-0.5">
                {selectedCount}
              </span>
            )}
            <ChevronDown className="ml-2 h-4 w-4 opacity-50" />
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-48 p-2" align="start">
          <div className="space-y-1">
            {config.options?.map((option) => (
              <button
                key={option.value}
                className="flex items-center gap-2 w-full rounded px-2 py-1.5 text-sm hover:bg-muted focus:outline-none focus:bg-muted"
                onClick={() => toggleOption(option.value)}
              >
                <div
                  className={cn(
                    "flex h-4 w-4 items-center justify-center rounded-sm border border-primary",
                    selected.includes(option.value)
                      ? "bg-primary text-primary-foreground"
                      : "opacity-50"
                  )}
                >
                  {selected.includes(option.value) && (
                    <Check className="h-3 w-3" />
                  )}
                </div>
                {option.label}
              </button>
            ))}
          </div>
        </PopoverContent>
      </Popover>
    );
  }

  return null;
}
