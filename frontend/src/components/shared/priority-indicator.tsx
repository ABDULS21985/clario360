import { cn } from "@/lib/utils";

type Priority = "P0" | "P1" | "P2" | "P3";

interface PriorityIndicatorProps {
  priority: Priority;
  showLabel?: boolean;
  size?: "sm" | "md" | "lg";
  className?: string;
}

const priorityConfig: Record<Priority, { label: string; color: string; dot: string }> = {
  P0: { label: "P0 Critical", color: "text-red-600 dark:text-red-400", dot: "bg-red-500" },
  P1: { label: "P1 High",     color: "text-orange-600 dark:text-orange-400", dot: "bg-orange-500" },
  P2: { label: "P2 Medium",   color: "text-yellow-600 dark:text-yellow-400", dot: "bg-yellow-500" },
  P3: { label: "P3 Low",      color: "text-blue-600 dark:text-blue-400", dot: "bg-blue-500" },
};

export function PriorityIndicator({ priority, showLabel = true, size = "md", className }: PriorityIndicatorProps) {
  const config = priorityConfig[priority] ?? priorityConfig.P3;
  const textSize = size === "sm" ? "text-xs" : size === "lg" ? "text-sm" : "text-xs";
  const dotSize = size === "sm" ? "h-1.5 w-1.5" : "h-2 w-2";

  return (
    <span className={cn("inline-flex items-center gap-1.5 font-medium", textSize, config.color, className)}>
      <span className={cn("rounded-full shrink-0", dotSize, config.dot)} aria-hidden />
      {showLabel && config.label}
    </span>
  );
}
