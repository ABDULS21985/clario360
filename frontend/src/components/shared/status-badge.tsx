import { type LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";
import type { StatusConfig } from "@/lib/status-configs";

interface StatusBadgeProps {
  status: string;
  config?: StatusConfig;
  variant?: "default" | "outline" | "dot";
  size?: "sm" | "md" | "lg";
  className?: string;
}

const colorMap: Record<string, { bg: string; text: string; border: string; dot: string }> = {
  red:    { bg: "bg-red-100 dark:bg-red-900/30",    text: "text-red-700 dark:text-red-400",    border: "border-red-300",    dot: "bg-red-500" },
  orange: { bg: "bg-orange-100 dark:bg-orange-900/30", text: "text-orange-700 dark:text-orange-400", border: "border-orange-300", dot: "bg-orange-500" },
  yellow: { bg: "bg-yellow-100 dark:bg-yellow-900/30", text: "text-yellow-700 dark:text-yellow-400", border: "border-yellow-300", dot: "bg-yellow-500" },
  green:  { bg: "bg-green-100 dark:bg-green-900/30",  text: "text-green-700 dark:text-green-400",  border: "border-green-300",  dot: "bg-green-500" },
  blue:   { bg: "bg-blue-100 dark:bg-blue-900/30",   text: "text-blue-700 dark:text-blue-400",   border: "border-blue-300",   dot: "bg-blue-500" },
  purple: { bg: "bg-purple-100 dark:bg-purple-900/30", text: "text-purple-700 dark:text-purple-400", border: "border-purple-300", dot: "bg-purple-500" },
  gray:   { bg: "bg-gray-100 dark:bg-gray-800",       text: "text-gray-600 dark:text-gray-400",    border: "border-gray-300",   dot: "bg-gray-400" },
  teal:   { bg: "bg-teal-100 dark:bg-teal-900/30",   text: "text-teal-700 dark:text-teal-400",   border: "border-teal-300",   dot: "bg-teal-500" },
};

const sizeClasses = {
  sm: "text-xs px-1.5 py-0.5 gap-1",
  md: "text-xs px-2 py-0.5 gap-1.5",
  lg: "text-sm px-2.5 py-1 gap-2",
};

const iconSizes = { sm: "h-3 w-3", md: "h-3.5 w-3.5", lg: "h-4 w-4" };

export function StatusBadge({ status, config, variant = "default", size = "md", className }: StatusBadgeProps) {
  const configItem = config?.[status];
  const color = configItem?.color ?? "gray";
  const label = configItem?.label ?? status;
  const Icon: LucideIcon | undefined = configItem?.icon;
  const colors = colorMap[color] ?? colorMap.gray;

  if (variant === "dot") {
    return (
      <span className={cn("inline-flex items-center gap-1.5", sizeClasses[size], className)}>
        <span className={cn("h-2 w-2 rounded-full shrink-0", colors.dot)} aria-hidden />
        <span className={colors.text}>{label}</span>
      </span>
    );
  }

  if (variant === "outline") {
    return (
      <span className={cn(
        "inline-flex items-center rounded-full border font-medium",
        sizeClasses[size], colors.text, colors.border, className
      )}>
        {Icon && <Icon className={cn(iconSizes[size], "shrink-0")} aria-hidden />}
        {label}
      </span>
    );
  }

  return (
    <span className={cn(
      "inline-flex items-center rounded-full font-medium",
      sizeClasses[size], colors.bg, colors.text, className
    )}>
      {Icon && <Icon className={cn(iconSizes[size], "shrink-0")} aria-hidden />}
      {label}
    </span>
  );
}
