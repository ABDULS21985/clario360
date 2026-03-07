'use client';

import { Plus } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import type { MapValuesTransformDraft } from '@/app/(dashboard)/data/pipelines/_components/pipeline-wizard-types';

interface MapTransformProps {
  value: MapValuesTransformDraft;
  availableColumns: string[];
  onChange: (value: MapValuesTransformDraft) => void;
}

function createMappingRow(): MapValuesTransformDraft['config']['mappings'][number] {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return { id: crypto.randomUUID(), key: '', value: '' };
  }
  return { id: Math.random().toString(36).slice(2, 10), key: '', value: '' };
}

export function MapTransform({
  value,
  availableColumns,
  onChange,
}: MapTransformProps) {
  return (
    <div className="space-y-4">
      <Select
        value={value.config.column}
        onValueChange={(next) => onChange({ ...value, config: { ...value.config, column: next } })}
      >
        <SelectTrigger>
          <SelectValue placeholder="Column" />
        </SelectTrigger>
        <SelectContent>
          {availableColumns.map((column) => (
            <SelectItem key={column} value={column}>
              {column}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <div className="space-y-3">
        {value.config.mappings.map((mapping) => (
          <div key={mapping.id} className="grid gap-3 rounded-lg border p-3 md:grid-cols-[1fr_auto_1fr_auto]">
            <Input
              value={mapping.key}
              onChange={(event) =>
                onChange({
                  ...value,
                  config: {
                    ...value.config,
                    mappings: value.config.mappings.map((item) =>
                      item.id === mapping.id ? { ...item, key: event.target.value } : item,
                    ),
                  },
                })
              }
              placeholder="Original value"
            />
            <div className="flex items-center justify-center text-sm text-muted-foreground">→</div>
            <Input
              value={mapping.value}
              onChange={(event) =>
                onChange({
                  ...value,
                  config: {
                    ...value.config,
                    mappings: value.config.mappings.map((item) =>
                      item.id === mapping.id ? { ...item, value: event.target.value } : item,
                    ),
                  },
                })
              }
              placeholder="Mapped value"
            />
            <Button
              type="button"
              variant="ghost"
              onClick={() =>
                onChange({
                  ...value,
                  config: {
                    ...value.config,
                    mappings: value.config.mappings.filter((item) => item.id !== mapping.id),
                  },
                })
              }
            >
              Remove
            </Button>
          </div>
        ))}
      </div>

      <div className="grid gap-4 md:grid-cols-[1fr_auto]">
        <Input
          value={value.config.default_value}
          onChange={(event) => onChange({ ...value, config: { ...value.config, default_value: event.target.value } })}
          placeholder="Default value for unmapped items"
        />
        <Button
          type="button"
          variant="outline"
          onClick={() =>
            onChange({
              ...value,
              config: {
                ...value.config,
                mappings: [...value.config.mappings, createMappingRow()],
              },
            })
          }
        >
          <Plus className="mr-2 h-4 w-4" />
          Add mapping
        </Button>
      </div>
    </div>
  );
}

