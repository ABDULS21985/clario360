"use client";
import { ArrowDown, ArrowUp, ArrowUpDown } from "lucide-react";
import type { Column } from "@tanstack/react-table";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";

interface DataTableColumnHeaderProps<TData, TValue> {
  column: Column<TData, TValue>;
  title: string;
  className?: string;
  onSortChange?: (columnId: string, direction: "asc" | "desc") => void;
  sortColumn?: string;
  sortDirection?: "asc" | "desc";
}

export function DataTableColumnHeader<TData, TValue>({
  column,
  title,
  className,
  onSortChange,
  sortColumn,
  sortDirection,
}: DataTableColumnHeaderProps<TData, TValue>) {
  if (!column.getCanSort()) {
    return <div className={cn("text-sm font-medium", className)}>{title}</div>;
  }

  const isCurrentSort = sortColumn === column.id;
  const currentDirection = isCurrentSort ? sortDirection : undefined;

  const handleSort = () => {
    if (!onSortChange) return;
    const nextDirection = currentDirection === "asc" ? "desc" : "asc";
    onSortChange(column.id, nextDirection);
  };

  const ariaSort = isCurrentSort
    ? currentDirection === "asc"
      ? "ascending"
      : "descending"
    : "none";

  return (
    <div className={cn("flex items-center gap-1", className)}>
      <Button
        variant="ghost"
        size="sm"
        className="-ml-3 h-8 rounded-xl px-3 text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground hover:text-foreground data-[state=open]:bg-accent"
        onClick={handleSort}
        aria-sort={ariaSort}
      >
        <span>{title}</span>
        {isCurrentSort ? (
          currentDirection === "desc" ? (
            <ArrowDown className="ml-1 h-4 w-4" />
          ) : (
            <ArrowUp className="ml-1 h-4 w-4" />
          )
        ) : (
          <ArrowUpDown className="ml-1 h-4 w-4 opacity-40" />
        )}
      </Button>
    </div>
  );
}
