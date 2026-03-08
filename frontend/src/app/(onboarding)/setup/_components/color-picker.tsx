'use client';

import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

export function ColorPicker({
  id,
  label,
  value,
  onChange,
  error,
}: {
  id: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  error?: string;
}) {
  return (
    <div className="space-y-2">
      <Label htmlFor={id}>{label}</Label>
      <div className="flex items-center gap-3">
        <input
          type="color"
          id={id}
          value={value}
          onChange={(event) => onChange(event.target.value)}
          className="h-11 w-14 rounded-md border border-slate-200 bg-white"
        />
        <Input value={value} onChange={(event) => onChange(event.target.value)} />
      </div>
      {error ? <p className="text-sm text-destructive">{error}</p> : null}
    </div>
  );
}
