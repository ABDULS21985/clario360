'use client';

import { Plus } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import type { AggregateTransformDraft } from '@/app/(dashboard)/data/pipelines/_components/pipeline-wizard-types';
import { createEmptyAggregation } from '@/app/(dashboard)/data/pipelines/_components/pipeline-wizard-utils';

interface AggregateTransformProps {
  value: AggregateTransformDraft;
  availableColumns: string[];
  onChange: (value: AggregateTransformDraft) => void;
}

const FUNCTIONS = ['count', 'sum', 'avg', 'min', 'max', 'count_distinct'] as const;

export function AggregateTransform({
  value,
  availableColumns,
  onChange,
}: AggregateTransformProps) {
  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <div className="text-sm font-medium">Group by</div>
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
          {availableColumns.map((column) => (
            <label key={column} className="flex items-center gap-3 rounded-md border px-3 py-2 text-sm">
              <Checkbox
                checked={value.config.group_by.includes(column)}
                onCheckedChange={() =>
                  onChange({
                    ...value,
                    config: {
                      ...value.config,
                      group_by: value.config.group_by.includes(column)
                        ? value.config.group_by.filter((item) => item !== column)
                        : [...value.config.group_by, column],
                    },
                  })
                }
              />
              <span>{column}</span>
            </label>
          ))}
        </div>
      </div>

      <div className="space-y-3">
        {value.config.aggregations.map((aggregation) => (
          <div key={aggregation.id} className="grid grid-cols-1 gap-3 rounded-lg border p-3 lg:grid-cols-[1fr_180px_1fr_auto]">
            <Select
              value={aggregation.column}
              onValueChange={(next) =>
                onChange({
                  ...value,
                  config: {
                    ...value.config,
                    aggregations: value.config.aggregations.map((item) =>
                      item.id === aggregation.id ? { ...item, column: next } : item,
                    ),
                  },
                })
              }
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

            <Select
              value={aggregation.function}
              onValueChange={(next) =>
                onChange({
                  ...value,
                  config: {
                    ...value.config,
                    aggregations: value.config.aggregations.map((item) =>
                      item.id === aggregation.id
                        ? {
                            ...item,
                            function: next as AggregateTransformDraft['config']['aggregations'][number]['function'],
                          }
                        : item,
                    ),
                  },
                })
              }
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {FUNCTIONS.map((fn) => (
                  <SelectItem key={fn} value={fn}>
                    {fn}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            <Input
              value={aggregation.alias}
              onChange={(event) =>
                onChange({
                  ...value,
                  config: {
                    ...value.config,
                    aggregations: value.config.aggregations.map((item) =>
                      item.id === aggregation.id ? { ...item, alias: event.target.value } : item,
                    ),
                  },
                })
              }
              placeholder="Alias"
            />

            <Button
              type="button"
              variant="ghost"
              onClick={() =>
                onChange({
                  ...value,
                  config: {
                    ...value.config,
                    aggregations: value.config.aggregations.filter((item) => item.id !== aggregation.id),
                  },
                })
              }
            >
              Remove
            </Button>
          </div>
        ))}
      </div>

      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={() =>
          onChange({
            ...value,
            config: {
              ...value.config,
              aggregations: [...value.config.aggregations, createEmptyAggregation()],
            },
          })
        }
      >
        <Plus className="mr-2 h-4 w-4" />
        Add aggregation
      </Button>
    </div>
  );
}

