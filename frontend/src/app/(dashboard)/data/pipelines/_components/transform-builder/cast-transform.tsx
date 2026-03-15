'use client';

import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import type { CastTransformDraft } from '@/app/(dashboard)/data/pipelines/_components/pipeline-wizard-types';

interface CastTransformProps {
  value: CastTransformDraft;
  availableColumns: string[];
  onChange: (value: CastTransformDraft) => void;
}

const TARGET_TYPES = ['string', 'integer', 'float', 'boolean', 'datetime'] as const;

export function CastTransform({
  value,
  availableColumns,
  onChange,
}: CastTransformProps) {
  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
      <div className="space-y-1.5">
        <div className="text-sm font-medium">Column</div>
        <Select
          value={value.config.column}
          onValueChange={(next) => onChange({ ...value, config: { ...value.config, column: next } })}
        >
          <SelectTrigger>
            <SelectValue placeholder="Select column" />
          </SelectTrigger>
          <SelectContent>
            {availableColumns.map((column) => (
              <SelectItem key={column} value={column}>
                {column}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="space-y-1.5">
        <div className="text-sm font-medium">Target type</div>
        <Select
          value={value.config.to_type}
          onValueChange={(next) =>
            onChange({
              ...value,
              config: { ...value.config, to_type: next as CastTransformDraft['config']['to_type'] },
            })
          }
        >
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {TARGET_TYPES.map((type) => (
              <SelectItem key={type} value={type}>
                {type}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  );
}
