'use client';

import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import type { RenameTransformDraft } from '@/app/(dashboard)/data/pipelines/_components/pipeline-wizard-types';

interface RenameTransformProps {
  value: RenameTransformDraft;
  availableColumns: string[];
  onChange: (value: RenameTransformDraft) => void;
}

export function RenameTransform({
  value,
  availableColumns,
  onChange,
}: RenameTransformProps) {
  return (
    <div className="grid gap-4 md:grid-cols-2">
      <div className="space-y-1.5">
        <div className="text-sm font-medium">From</div>
        <Select
          value={value.config.from}
          onValueChange={(next) => onChange({ ...value, config: { ...value.config, from: next } })}
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
        <div className="text-sm font-medium">To</div>
        <Input
          value={value.config.to}
          onChange={(event) => onChange({ ...value, config: { ...value.config, to: event.target.value } })}
          placeholder="renamed_column"
        />
      </div>
    </div>
  );
}
