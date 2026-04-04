import { AlertTriangle, AlertCircle, Info, ArrowDown, type LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";

export type Severity =
  | "critical"
  | "high"
  | "warning"
  | "medium"
  | "low"
  | "info";

interface SeverityIndicatorProps {
  severity: Severity;
  showLabel?: boolean;
  showIcon?: boolean;
  size?: "sm" | "md" | "lg";
  className?: string;
}

const severityConfig: Record<Severity, { label: string; icon: LucideIcon; bg: string; text: string }> = {
  critical: { label: "Critical", icon: AlertTriangle, bg: "bg-red-100 dark:bg-red-900/30", text: "text-red-600 dark:text-red-400" },
  high:     { label: "High",     icon: AlertCircle,   bg: "bg-orange-100 dark:bg-orange-900/30", text: "text-orange-600 dark:text-orange-400" },
  warning:  { label: "Warning",  icon: AlertCircle,   bg: "bg-yellow-100 dark:bg-yellow-900/30", text: "text-yellow-600 dark:text-yellow-400" },
  medium:   { label: "Medium",   icon: Info,          bg: "bg-yellow-100 dark:bg-yellow-900/30", text: "text-yellow-600 dark:text-yellow-400" },
  low:      { label: "Low",      icon: ArrowDown,     bg: "bg-blue-100 dark:bg-blue-900/30", text: "text-blue-600 dark:text-blue-400" },
  info:     { label: "Info",     icon: Info,          bg: "bg-gray-100 dark:bg-gray-800", text: "text-gray-500 dark:text-gray-400" },
};

const sizeClasses = { sm: "text-xs px-1.5 py-0.5 gap-1", md: "text-xs px-2 py-0.5 gap-1.5", lg: "text-sm px-2.5 py-1 gap-2" };
const iconSizes = { sm: "h-3 w-3", md: "h-3.5 w-3.5", lg: "h-4 w-4" };

export function SeverityIndicator({
  severity,
  showLabel = true,
  showIcon = true,
  size = "md",
  className,
}: SeverityIndicatorProps) {
  const config = severityConfig[severity] ?? severityConfig.info;
  const Icon = config.icon;

  return (
    <span className={cn(
      "inline-flex items-center rounded-full font-medium",
      sizeClasses[size], config.bg, config.text, className
    )}>
      {showIcon && <Icon className={cn(iconSizes[size], "shrink-0")} aria-hidden />}
      {showLabel && config.label}
    </span>
  );
}
