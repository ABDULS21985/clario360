'use client';

import { Checkbox } from '@/components/ui/checkbox';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import type { DeduplicateTransformDraft } from '@/app/(dashboard)/data/pipelines/_components/pipeline-wizard-types';

interface DedupTransformProps {
  value: DeduplicateTransformDraft;
  availableColumns: string[];
  onChange: (value: DeduplicateTransformDraft) => void;
}

export function DedupTransform({
  value,
  availableColumns,
  onChange,
}: DedupTransformProps) {
  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <div className="text-sm font-medium">Key columns</div>
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
          {availableColumns.map((column) => (
            <label key={column} className="flex items-center gap-3 rounded-md border px-3 py-2 text-sm">
              <Checkbox
                checked={value.config.key_columns.includes(column)}
                onCheckedChange={() =>
                  onChange({
                    ...value,
                    config: {
                      ...value.config,
                      key_columns: value.config.key_columns.includes(column)
                        ? value.config.key_columns.filter((item) => item !== column)
                        : [...value.config.key_columns, column],
                    },
                  })
                }
              />
              <span>{column}</span>
            </label>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <Select
          value={value.config.keep}
          onValueChange={(next) =>
            onChange({
              ...value,
              config: { ...value.config, keep: next as DeduplicateTransformDraft['config']['keep'] },
            })
          }
        >
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="latest">Latest</SelectItem>
            <SelectItem value="first">First</SelectItem>
          </SelectContent>
        </Select>

        <Select
          value={value.config.order_by}
          onValueChange={(next) => onChange({ ...value, config: { ...value.config, order_by: next } })}
        >
          <SelectTrigger>
            <SelectValue placeholder="Order by column" />
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
    </div>
  );
}

