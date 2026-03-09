"use client";
import { useState } from "react";
import { Download, Settings2, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { Column } from "@tanstack/react-table";
import { DataTableFilter } from "./data-table-filter";
import { DataTableActiveFilters } from "./data-table-active-filters";
import type { FilterConfig, BulkAction } from "@/types/table";

interface DataTableToolbarProps<TData> {
  searchSlot?: React.ReactNode; // pass in SearchInput component
  filters?: FilterConfig[];
  activeFilters?: Record<string, string | string[]>;
  onFilterChange?: (key: string, value: string | string[] | undefined) => void;
  onClearFilters?: () => void;
  columns?: Column<TData>[];
  columnVisibility?: Record<string, boolean>;
  onColumnVisibilityChange?: (visibility: Record<string, boolean>) => void;
  enableColumnToggle?: boolean;
  enableExport?: boolean;
  onExport?: (format: "csv" | "json") => void;
  selectedCount?: number;
  bulkActions?: BulkAction[];
  getSelectedIds?: () => string[];
}

export function DataTableToolbar<TData>({
  searchSlot,
  filters,
  activeFilters = {},
  onFilterChange,
  onClearFilters,
  columns,
  columnVisibility = {},
  onColumnVisibilityChange,
  enableColumnToggle = true,
  enableExport = false,
  onExport,
  selectedCount = 0,
  bulkActions,
  getSelectedIds,
}: DataTableToolbarProps<TData>) {
  const [bulkLoading, setBulkLoading] = useState<string | null>(null);

  const hasActiveFilters = Object.values(activeFilters).some(
    (v) =>
      v !== undefined && v !== "" && !(Array.isArray(v) && v.length === 0)
  );

  const handleBulkAction = async (action: BulkAction) => {
    const ids = getSelectedIds?.() ?? [];
    setBulkLoading(action.label);
    try {
      await action.onClick(ids);
    } finally {
      setBulkLoading(null);
    }
  };

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2 rounded-[22px] border border-[color:var(--card-border)] bg-[rgba(255,255,255,0.74)] p-3 shadow-[0_18px_40px_-36px_rgba(15,23,42,0.3)] backdrop-blur-md">
        {searchSlot && (
          <div className="min-w-[220px] flex-1 max-w-md">{searchSlot}</div>
        )}

        {filters?.map((filter) => (
          <DataTableFilter
            key={filter.key}
            config={filter}
            value={activeFilters[filter.key]}
            onChange={onFilterChange ?? (() => {})}
          />
        ))}

        {hasActiveFilters && (
          <Button
            variant="ghost"
            size="sm"
            className="h-9 px-3 text-muted-foreground"
            onClick={onClearFilters}
          >
            <X className="mr-1 h-4 w-4" />
            Clear
          </Button>
        )}

        <div className="ml-auto flex items-center gap-2">
          {enableColumnToggle && columns && onColumnVisibilityChange && (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="sm" className="h-9">
                  <Settings2 className="mr-2 h-4 w-4" />
                  Columns
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-48">
                <DropdownMenuLabel>Toggle columns</DropdownMenuLabel>
                <DropdownMenuSeparator />
                {columns
                  .filter((col) => col.getCanHide())
                  .map((col) => (
                    <DropdownMenuCheckboxItem
                      key={col.id}
                      checked={columnVisibility[col.id] !== false}
                      onCheckedChange={(checked) =>
                        onColumnVisibilityChange({
                          ...columnVisibility,
                          [col.id]: checked,
                        })
                      }
                    >
                      {col.id}
                    </DropdownMenuCheckboxItem>
                  ))}
              </DropdownMenuContent>
            </DropdownMenu>
          )}

          {enableExport && onExport && (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="sm" className="h-9">
                  <Download className="mr-2 h-4 w-4" />
                  Export
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuLabel>Export as</DropdownMenuLabel>
                <DropdownMenuSeparator />
                <button
                  className="w-full px-2 py-1.5 text-sm text-left hover:bg-muted rounded"
                  onClick={() => onExport("csv")}
                >
                  CSV
                </button>
                <button
                  className="w-full px-2 py-1.5 text-sm text-left hover:bg-muted rounded"
                  onClick={() => onExport("json")}
                >
                  JSON
                </button>
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </div>
      </div>

      {hasActiveFilters && filters && (
        <DataTableActiveFilters
          activeFilters={activeFilters}
          filterConfigs={filters}
          onRemoveFilter={(key) => onFilterChange?.(key, undefined)}
          onClearAll={onClearFilters ?? (() => {})}
        />
      )}

      {selectedCount > 0 && bulkActions && bulkActions.length > 0 && (
        <div className="flex items-center gap-2 rounded-[20px] border border-primary/10 bg-primary/5 px-3 py-2.5">
          <span className="text-sm font-medium">{selectedCount} selected</span>
          <div className="ml-auto flex items-center gap-2">
            {bulkActions.map((action) => (
              <Button
                key={action.label}
                variant={
                  action.variant === "destructive" ? "destructive" : "outline"
                }
                size="sm"
                className="h-8"
                onClick={() => handleBulkAction(action)}
                disabled={bulkLoading !== null}
              >
                {action.icon && <action.icon className="mr-2 h-4 w-4" />}
                {bulkLoading === action.label ? "Processing..." : action.label}
              </Button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
