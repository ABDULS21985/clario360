import type { LucideIcon } from "lucide-react";
import { InboxIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface DataTableEmptyProps {
  icon?: LucideIcon;
  title?: string;
  description?: string;
  action?: { label: string; onClick: () => void; icon?: LucideIcon };
  hasActiveFilters?: boolean;
  className?: string;
}

export function DataTableEmpty({
  icon: Icon = InboxIcon,
  title = "No results found",
  description,
  action,
  hasActiveFilters = false,
  className,
}: DataTableEmptyProps) {
  const defaultDescription = hasActiveFilters
    ? "No results match your current filters. Try adjusting or clearing your filters."
    : "No data available yet.";

  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center py-16 text-center",
        className
      )}
      role="status"
      aria-live="polite"
    >
      <Icon className="h-12 w-12 text-muted-foreground/40 mb-4" />
      <h3 className="text-sm font-semibold text-foreground mb-1">{title}</h3>
      <p className="text-sm text-muted-foreground max-w-sm">
        {description ?? defaultDescription}
      </p>
      {action && (
        <Button
          variant="outline"
          size="sm"
          className="mt-4"
          onClick={action.onClick}
        >
          {action.icon && <action.icon className="mr-2 h-4 w-4" />}
          {action.label}
        </Button>
      )}
    </div>
  );
}
