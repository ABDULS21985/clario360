import { AlertCircle, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface DataTableErrorProps {
  error: string;
  onRetry?: () => void;
  className?: string;
}

export function DataTableError({
  error,
  onRetry,
  className,
}: DataTableErrorProps) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center py-16 text-center",
        className
      )}
      role="alert"
      aria-live="assertive"
    >
      <AlertCircle className="h-12 w-12 text-destructive/60 mb-4" />
      <h3 className="text-sm font-semibold text-foreground mb-1">
        Failed to load data
      </h3>
      <p className="text-sm text-muted-foreground max-w-sm mb-4">{error}</p>
      {onRetry && (
        <Button variant="outline" size="sm" onClick={onRetry}>
          <RefreshCw className="mr-2 h-4 w-4" />
          Retry
        </Button>
      )}
    </div>
  );
}
