"use client";
import { useEffect, useState } from "react";
import { Check, ChevronDown } from "lucide-react";
import { Button } from "@/components/ui/button";
import { DateRangePicker } from "@/components/shared/forms/date-range-picker";
import { Slider } from "@/components/ui/slider";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import type { FilterConfig } from "@/types/table";

interface DataTableFilterProps {
  config: FilterConfig;
  value: string | string[] | undefined;
  onChange: (key: string, value: string | string[] | undefined) => void;
}

export function DataTableFilter({
  config,
  value,
  onChange,
}: DataTableFilterProps) {
  const [open, setOpen] = useState(false);

  const isActive =
    value !== undefined &&
    value !== "" &&
    !(Array.isArray(value) && value.length === 0);
  const selectedCount = Array.isArray(value)
    ? value.length
    : isActive
      ? 1
      : 0;

  if (config.type === "date-range") {
    const [fromValue, toValue] = typeof value === "string" ? value.split(",") : [];
    const range = {
      from: fromValue ? new Date(fromValue) : undefined,
      to: toValue ? new Date(toValue) : undefined,
    };

    return (
      <DateRangePicker
        value={range}
        onChange={(nextRange) => {
          if (!nextRange.from && !nextRange.to) {
            onChange(config.key, undefined);
            return;
          }

          const serializedFrom = nextRange.from
            ? nextRange.from.toISOString()
            : "";
          const serializedTo = nextRange.to ? nextRange.to.toISOString() : "";
          onChange(config.key, `${serializedFrom},${serializedTo}`);
        }}
        className={cn("h-8", isActive && "border-primary")}
      />
    );
  }

  if (config.type === "select" || config.type === "multi-select") {
    const isMulti = config.type === "multi-select";
    const selected = Array.isArray(value) ? value : value ? [value] : [];

    const toggleOption = (optValue: string) => {
      if (isMulti) {
        const next = selected.includes(optValue)
          ? selected.filter((v) => v !== optValue)
          : [...selected, optValue];
        onChange(config.key, next.length > 0 ? next : undefined);
      } else {
        onChange(
          config.key,
          selected[0] === optValue ? undefined : optValue
        );
        setOpen(false);
      }
    };

    return (
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            size="sm"
            className={cn("h-8 border-dashed", isActive && "border-primary")}
          >
            {config.label}
            {selectedCount > 0 && (
              <span className="ml-2 rounded-full bg-primary text-primary-foreground text-xs px-1.5 py-0.5">
                {selectedCount}
              </span>
            )}
            <ChevronDown className="ml-2 h-4 w-4 opacity-50" />
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-48 p-2" align="start">
          <div className="space-y-1">
            {config.options?.map((option) => (
              <button
                key={option.value}
                className="flex items-center gap-2 w-full rounded px-2 py-1.5 text-sm hover:bg-muted focus:outline-none focus:bg-muted"
                onClick={() => toggleOption(option.value)}
              >
                <div
                  className={cn(
                    "flex h-4 w-4 items-center justify-center rounded-sm border border-primary",
                    selected.includes(option.value)
                      ? "bg-primary text-primary-foreground"
                      : "opacity-50"
                  )}
                >
                  {selected.includes(option.value) && (
                    <Check className="h-3 w-3" />
                  )}
                </div>
                {option.label}
              </button>
            ))}
          </div>
        </PopoverContent>
      </Popover>
    );
  }

  if (config.type === "text") {
    return (
      <TextFilterControl
        config={config}
        isActive={isActive}
        open={open}
        setOpen={setOpen}
        value={value}
        onChange={onChange}
      />
    );
  }

  if (config.type === "range") {
    return (
      <RangeFilterControl
        config={config}
        isActive={isActive}
        open={open}
        setOpen={setOpen}
        value={value}
        onChange={onChange}
      />
    );
  }

  return null;
}

interface TextFilterControlProps {
  config: FilterConfig;
  isActive: boolean;
  open: boolean;
  setOpen: (open: boolean) => void;
  value: string | string[] | undefined;
  onChange: (key: string, value: string | string[] | undefined) => void;
}

function TextFilterControl({
  config,
  isActive,
  open,
  setOpen,
  value,
  onChange,
}: TextFilterControlProps) {
  const current = typeof value === "string" ? value : "";
  const [local, setLocal] = useState(current);

  useEffect(() => {
    setLocal(typeof value === "string" ? value : "");
  }, [value]);

  const apply = () => {
    onChange(config.key, local.trim() || undefined);
    setOpen(false);
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          size="sm"
          className={cn("h-8 border-dashed", isActive && "border-primary")}
        >
          {config.label}
          {isActive && (
            <span className="ml-2 max-w-[80px] truncate text-xs text-muted-foreground">
              {current}
            </span>
          )}
          <ChevronDown className="ml-2 h-4 w-4 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-64 space-y-3 p-3" align="start">
        <input
          className="w-full rounded-md border px-2 py-1.5 text-sm outline-none focus:ring-1 focus:ring-primary"
          placeholder={config.placeholder ?? `Filter by ${config.label.toLowerCase()}…`}
          value={local}
          onChange={(e) => setLocal(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") apply();
          }}
          autoFocus
        />
        <div className="flex justify-end gap-2">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => {
              setLocal("");
              onChange(config.key, undefined);
              setOpen(false);
            }}
          >
            Clear
          </Button>
          <Button type="button" size="sm" onClick={apply}>
            Apply
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  );
}

interface RangeFilterControlProps {
  config: FilterConfig;
  isActive: boolean;
  open: boolean;
  setOpen: (open: boolean) => void;
  value: string | string[] | undefined;
  onChange: (key: string, value: string | string[] | undefined) => void;
}

function RangeFilterControl({
  config,
  isActive,
  open,
  setOpen,
  value,
  onChange,
}: RangeFilterControlProps) {
  const min = config.min ?? 0;
  const max = config.max ?? 100;
  const step = config.step ?? 1;
  const suffix = config.valueSuffix ?? "";
  const parsed = typeof value === "string"
    ? value.split(",").map((part) => Number(part))
    : [min, max];
  const initialRange: [number, number] = [
    Number.isFinite(parsed[0]) ? parsed[0] : min,
    Number.isFinite(parsed[1]) ? parsed[1] : max,
  ];
  const [range, setRange] = useState<[number, number]>(initialRange);

  useEffect(() => {
    setRange(initialRange);
  }, [initialRange[0], initialRange[1]]);

  const displayValue = `${range[0]}${suffix} - ${range[1]}${suffix}`;

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          size="sm"
          className={cn("h-8 border-dashed", isActive && "border-primary")}
        >
          {config.label}
          {isActive && (
            <span className="ml-2 text-xs text-muted-foreground">
              {displayValue}
            </span>
          )}
          <ChevronDown className="ml-2 h-4 w-4 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-72 space-y-4 p-4" align="start">
        <div className="flex items-center justify-between text-sm">
          <span>{range[0]}{suffix}</span>
          <span>{range[1]}{suffix}</span>
        </div>
        <Slider
          value={range}
          min={min}
          max={max}
          step={step}
          onValueChange={(next) => {
            if (next.length === 2) {
              setRange([next[0] ?? min, next[1] ?? max]);
            }
          }}
        />
        <div className="flex items-center justify-between">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => {
              setRange([min, max]);
              onChange(config.key, undefined);
              setOpen(false);
            }}
          >
            Reset
          </Button>
          <Button
            type="button"
            size="sm"
            onClick={() => {
              const nextValue = range[0] <= min && range[1] >= max
                ? undefined
                : `${range[0]},${range[1]}`;
              onChange(config.key, nextValue);
              setOpen(false);
            }}
          >
            Apply
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  );
}
