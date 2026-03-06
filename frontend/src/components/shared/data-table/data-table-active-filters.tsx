"use client";
import { X } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import type { FilterConfig } from "@/types/table";

interface DataTableActiveFiltersProps {
  activeFilters: Record<string, string | string[]>;
  filterConfigs: FilterConfig[];
  onRemoveFilter: (key: string) => void;
  onClearAll: () => void;
}

export function DataTableActiveFilters({
  activeFilters,
  filterConfigs,
  onRemoveFilter,
  onClearAll,
}: DataTableActiveFiltersProps) {
  const entries = Object.entries(activeFilters).filter(
    ([, v]) =>
      v !== undefined && v !== "" && !(Array.isArray(v) && v.length === 0)
  );

  if (entries.length === 0) return null;

  const getLabel = (key: string, value: string | string[]): string => {
    const config = filterConfigs.find((f) => f.key === key);
    const displayValue = Array.isArray(value) ? value.join(", ") : value;
    if (!config) return `${key}: ${displayValue}`;
    if (config.options) {
      const labels = (Array.isArray(value) ? value : [value])
        .map((v) => config.options?.find((o) => o.value === v)?.label ?? v)
        .join(", ");
      return `${config.label}: ${labels}`;
    }
    return `${config.label}: ${displayValue}`;
  };

  return (
    <div className="flex flex-wrap items-center gap-2 py-2">
      <span className="text-xs text-muted-foreground font-medium">
        Active filters:
      </span>
      {entries.map(([key, value]) => (
        <Badge
          key={key}
          variant="secondary"
          className="gap-1 pr-1 text-xs"
        >
          {getLabel(key, value)}
          <button
            onClick={() => onRemoveFilter(key)}
            className="ml-1 rounded-full hover:bg-muted p-0.5 focus:outline-none focus:ring-1 focus:ring-ring"
            aria-label={`Remove ${key} filter`}
          >
            <X className="h-3 w-3" />
          </button>
        </Badge>
      ))}
      <Button
        variant="ghost"
        size="sm"
        className="h-6 px-2 text-xs"
        onClick={onClearAll}
      >
        Clear all
      </Button>
    </div>
  );
}
