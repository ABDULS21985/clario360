'use client';

import { Button } from '@/components/ui/button';

interface PeriodSelectorProps {
  value: string;
  onChange: (period: string) => void;
  options?: string[];
}

const DEFAULT_OPTIONS = ['24h', '7d', '30d'];

export function PeriodSelector({ value, onChange, options = DEFAULT_OPTIONS }: PeriodSelectorProps) {
  return (
    <div className="flex gap-1">
      {options.map((opt) => (
        <Button
          key={opt}
          size="sm"
          variant={value === opt ? 'default' : 'outline'}
          onClick={() => onChange(opt)}
          className="text-xs px-3"
        >
          {opt}
        </Button>
      ))}
    </div>
  );
}
