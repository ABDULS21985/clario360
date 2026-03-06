"use client";
import { useState } from "react";
import { CalendarIcon } from "lucide-react";
import { format, subDays, startOfMonth, endOfMonth, subMonths } from "date-fns";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Calendar } from "@/components/ui/calendar";
import { cn } from "@/lib/utils";

interface DateRange {
  from: Date | undefined;
  to: Date | undefined;
}

interface DateRangePickerProps {
  value: DateRange;
  onChange: (range: DateRange) => void;
  presets?: Array<{ label: string; from: Date; to: Date }>;
  disabled?: boolean;
  className?: string;
}

const defaultPresets = [
  { label: "Last 24 hours", from: subDays(new Date(), 1), to: new Date() },
  { label: "Last 7 days", from: subDays(new Date(), 7), to: new Date() },
  { label: "Last 30 days", from: subDays(new Date(), 30), to: new Date() },
  { label: "Last 90 days", from: subDays(new Date(), 90), to: new Date() },
  { label: "This month", from: startOfMonth(new Date()), to: new Date() },
  { label: "Last month", from: startOfMonth(subMonths(new Date(), 1)), to: endOfMonth(subMonths(new Date(), 1)) },
];

export function DateRangePicker({
  value,
  onChange,
  presets = defaultPresets,
  disabled = false,
  className,
}: DateRangePickerProps) {
  const [open, setOpen] = useState(false);

  const displayText = value.from
    ? value.to
      ? `${format(value.from, "MMM d, yyyy")} – ${format(value.to, "MMM d, yyyy")}`
      : format(value.from, "MMM d, yyyy")
    : "Select date range";

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          className={cn("justify-start gap-2 font-normal", !value.from && "text-muted-foreground", className)}
          disabled={disabled}
        >
          <CalendarIcon className="h-4 w-4 shrink-0" aria-hidden />
          {displayText}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="flex w-auto gap-0 p-0" align="start">
        <div className="flex flex-col gap-1 border-r p-3 text-sm">
          {presets.map((preset) => (
            <button
              key={preset.label}
              className="rounded px-2 py-1 text-left hover:bg-muted focus:outline-none focus:bg-muted whitespace-nowrap"
              onClick={() => { onChange({ from: preset.from, to: preset.to }); setOpen(false); }}
            >
              {preset.label}
            </button>
          ))}
        </div>
        <Calendar
          mode="range"
          selected={{ from: value.from, to: value.to }}
          onSelect={(range) => {
            onChange({ from: range?.from, to: range?.to });
          }}
          numberOfMonths={2}
          initialFocus
        />
      </PopoverContent>
    </Popover>
  );
}
