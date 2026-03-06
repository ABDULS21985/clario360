"use client";
import { MoreHorizontal } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";
import type { RowAction } from "@/types/table";

interface DataTableRowActionsProps<TData> {
  row: TData;
  actions: RowAction<TData>[] | ((row: TData) => RowAction<TData>[]);
}

export function DataTableRowActions<TData>({
  row,
  actions,
}: DataTableRowActionsProps<TData>) {
  const resolvedActions =
    typeof actions === "function" ? actions(row) : actions;
  const visibleActions = resolvedActions.filter((a) => !a.hidden?.(row));

  if (visibleActions.length === 0) return null;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8 p-0 data-[state=open]:bg-muted"
          onClick={(e) => e.stopPropagation()}
          aria-label="Row actions"
        >
          <MoreHorizontal className="h-4 w-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-48">
        {visibleActions.map((action, idx) => {
          const isDisabled = action.disabled?.(row) ?? false;
          const isDestructive = action.variant === "destructive";
          const prevAction = visibleActions[idx - 1];
          const showSeparator =
            prevAction?.variant !== action.variant && idx > 0;

          return (
            <div key={action.label}>
              {showSeparator && <DropdownMenuSeparator />}
              <DropdownMenuItem
                disabled={isDisabled}
                onClick={(e) => {
                  e.stopPropagation();
                  action.onClick(row);
                }}
                className={cn(
                  isDestructive && "text-destructive focus:text-destructive"
                )}
              >
                {action.icon && <action.icon className="mr-2 h-4 w-4" />}
                {action.label}
              </DropdownMenuItem>
            </div>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
