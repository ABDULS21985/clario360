"use client";
import { RefreshCw, AlertCircle, BarChart2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

interface ChartContainerProps {
  loading?: boolean;
  error?: string;
  onRetry?: () => void;
  empty?: boolean;
  emptyMessage?: string;
  height?: number;
  title?: string;
  subtitle?: string;
  children: React.ReactNode;
  className?: string;
}

export function ChartContainer({
  loading = false,
  error,
  onRetry,
  empty = false,
  emptyMessage = "No data available",
  height = 300,
  title,
  subtitle,
  children,
  className,
}: ChartContainerProps) {
  return (
    <div className={cn("w-full", className)}>
      {(title || subtitle) && (
        <div className="mb-3">
          {title && <h3 className="text-sm font-semibold">{title}</h3>}
          {subtitle && <p className="text-xs text-muted-foreground">{subtitle}</p>}
        </div>
      )}
      <div style={{ height }} className="w-full">
        {loading ? (
          <Skeleton className="w-full h-full rounded-lg" />
        ) : error ? (
          <div className="flex flex-col items-center justify-center h-full gap-2 text-center" role="alert">
            <AlertCircle className="h-8 w-8 text-destructive/60" aria-hidden />
            <p className="text-sm text-muted-foreground">{error}</p>
            {onRetry && (
              <Button variant="outline" size="sm" onClick={onRetry}>
                <RefreshCw className="mr-2 h-4 w-4" /> Retry
              </Button>
            )}
          </div>
        ) : empty ? (
          <div className="flex flex-col items-center justify-center h-full gap-2 text-center" role="status" aria-live="polite">
            <BarChart2 className="h-8 w-8 text-muted-foreground/40" aria-hidden />
            <p className="text-sm text-muted-foreground">{emptyMessage}</p>
          </div>
        ) : (
          children
        )}
      </div>
    </div>
  );
}
