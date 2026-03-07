'use client';

import { Input } from '@/components/ui/input';
import type { DeriveTransformDraft } from '@/app/(dashboard)/data/pipelines/_components/pipeline-wizard-types';

interface DeriveTransformProps {
  value: DeriveTransformDraft;
  availableColumns: string[];
  onChange: (value: DeriveTransformDraft) => void;
}

export function DeriveTransform({
  value,
  availableColumns,
  onChange,
}: DeriveTransformProps) {
  return (
    <div className="space-y-4">
      <div className="grid gap-4 md:grid-cols-2">
        <div className="space-y-1.5">
          <div className="text-sm font-medium">New column name</div>
          <Input
            value={value.config.name}
            onChange={(event) => onChange({ ...value, config: { ...value.config, name: event.target.value } })}
            placeholder="full_name"
          />
        </div>

        <div className="rounded-lg border bg-muted/20 p-3 text-xs text-muted-foreground">
          Functions: `UPPER`, `LOWER`, `TRIM`, `CONCAT`, `COALESCE`
        </div>
      </div>

      <div className="space-y-1.5">
        <div className="text-sm font-medium">Expression</div>
        <div className="text-xs text-muted-foreground">
          Available columns: {availableColumns.join(', ') || 'No columns selected yet'}
        </div>
        <Input
          value={value.config.expression}
          onChange={(event) => onChange({ ...value, config: { ...value.config, expression: event.target.value } })}
          placeholder="TRIM(first_name) + ' ' + TRIM(last_name)"
        />
      </div>
    </div>
  );
}
