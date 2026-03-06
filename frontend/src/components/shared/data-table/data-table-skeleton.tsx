import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

interface DataTableSkeletonProps {
  columns: number;
  rows?: number;
  hasCheckbox?: boolean;
  hasActions?: boolean;
  className?: string;
}

export function DataTableSkeleton({
  columns,
  rows = 10,
  hasCheckbox = false,
  hasActions = false,
  className,
}: DataTableSkeletonProps) {
  return (
    <div
      className={cn("w-full space-y-2", className)}
      aria-busy="true"
      aria-label="Loading table data"
    >
      {Array.from({ length: rows }).map((_, rowIndex) => (
        <div
          key={rowIndex}
          className="flex items-center gap-4 px-4 py-3 border-b border-border last:border-0"
        >
          {hasCheckbox && <Skeleton className="h-4 w-4 shrink-0 rounded" />}
          {Array.from({ length: columns }).map((_, colIndex) => (
            <Skeleton
              key={colIndex}
              className="h-4 rounded"
              style={{
                width: `${60 + ((rowIndex * 7 + colIndex * 13) % 30)}%`,
              }}
            />
          ))}
          {hasActions && (
            <Skeleton className="h-8 w-8 shrink-0 rounded ml-auto" />
          )}
        </div>
      ))}
    </div>
  );
}
