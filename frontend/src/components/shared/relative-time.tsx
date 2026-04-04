"use client";
import { useState, useEffect } from "react";
import { formatDistanceToNow, format } from "date-fns";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

interface RelativeTimeProps {
  date: string | Date;
  className?: string;
}

export function RelativeTime({ date, className }: RelativeTimeProps) {
  const dateObj = typeof date === "string" ? new Date(date) : date;
  const isValid = dateObj instanceof Date && !isNaN(dateObj.getTime());

  const [relative, setRelative] = useState(() =>
    isValid ? formatDistanceToNow(dateObj, { addSuffix: true }) : "—"
  );

  useEffect(() => {
    if (!isValid) return;
    const update = () => setRelative(formatDistanceToNow(dateObj, { addSuffix: true }));
    const interval = setInterval(update, 60_000);
    return () => clearInterval(interval);
  }, [dateObj, isValid]);

  if (!isValid) {
    return <span className={cn("text-sm text-muted-foreground", className)}>—</span>;
  }

  const fullDate = format(dateObj, "MMM d, yyyy 'at' HH:mm:ss 'UTC'");

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <time dateTime={dateObj.toISOString()} className={cn("cursor-default text-sm", className)}>
            {relative}
          </time>
        </TooltipTrigger>
        <TooltipContent>
          <p>{fullDate}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
