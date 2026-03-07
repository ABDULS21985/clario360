'use client';

import { Plus } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import type { FilterTransformDraft, FilterOperator } from '@/app/(dashboard)/data/pipelines/_components/pipeline-wizard-types';
import { createEmptyFilterCondition } from '@/app/(dashboard)/data/pipelines/_components/pipeline-wizard-utils';

interface FilterTransformProps {
  value: FilterTransformDraft;
  availableColumns: string[];
  onChange: (value: FilterTransformDraft) => void;
}

const OPERATORS: Array<{ label: string; value: FilterOperator }> = [
  { label: '==', value: '==' },
  { label: '!=', value: '!=' },
  { label: '>', value: '>' },
  { label: '<', value: '<' },
  { label: '>=', value: '>=' },
  { label: '<=', value: '<=' },
  { label: 'in', value: 'in' },
  { label: 'not_in', value: 'not_in' },
  { label: 'like', value: 'like' },
  { label: 'is_null', value: 'is_null' },
  { label: 'is_not_null', value: 'is_not_null' },
];

export function FilterTransform({
  value,
  availableColumns,
  onChange,
}: FilterTransformProps) {
  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <span className="text-sm font-medium">Combine with</span>
        <Select
          value={value.config.combinator}
          onValueChange={(next) =>
            onChange({ ...value, config: { ...value.config, combinator: next as 'AND' | 'OR' } })
          }
        >
          <SelectTrigger className="w-32">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="AND">AND</SelectItem>
            <SelectItem value="OR">OR</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="space-y-3">
        {value.config.conditions.map((condition) => (
          <div key={condition.id} className="grid gap-3 rounded-lg border p-3 lg:grid-cols-[1fr_140px_1fr_auto]">
            <Select
              value={condition.column}
              onValueChange={(next) =>
                onChange({
                  ...value,
                  config: {
                    ...value.config,
                    conditions: value.config.conditions.map((item) =>
                      item.id === condition.id ? { ...item, column: next } : item,
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
              value={condition.operator}
              onValueChange={(next) =>
                onChange({
                  ...value,
                  config: {
                    ...value.config,
                    conditions: value.config.conditions.map((item) =>
                      item.id === condition.id ? { ...item, operator: next as FilterOperator } : item,
                    ),
                  },
                })
              }
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {OPERATORS.map((operator) => (
                  <SelectItem key={operator.value} value={operator.value}>
                    {operator.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            {condition.operator === 'is_null' || condition.operator === 'is_not_null' ? (
              <div className="flex items-center rounded-md border px-3 text-sm text-muted-foreground">
                No value required
              </div>
            ) : condition.operator === 'in' || condition.operator === 'not_in' ? (
              <Input
                value={condition.value}
                onChange={(event) =>
                  onChange({
                    ...value,
                    config: {
                      ...value.config,
                      conditions: value.config.conditions.map((item) =>
                        item.id === condition.id ? { ...item, value: event.target.value } : item,
                      ),
                    },
                  })
                }
                placeholder="A, B, C"
              />
            ) : (
              <Input
                value={condition.value}
                onChange={(event) =>
                  onChange({
                    ...value,
                    config: {
                      ...value.config,
                      conditions: value.config.conditions.map((item) =>
                        item.id === condition.id ? { ...item, value: event.target.value } : item,
                      ),
                    },
                  })
                }
                placeholder="Value"
              />
            )}

            <Button
              type="button"
              variant="ghost"
              onClick={() =>
                onChange({
                  ...value,
                  config: {
                    ...value.config,
                    conditions: value.config.conditions.filter((item) => item.id !== condition.id),
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
              conditions: [...value.config.conditions, createEmptyFilterCondition()],
            },
          })
        }
      >
        <Plus className="mr-2 h-4 w-4" />
        Add condition
      </Button>
    </div>
  );
}
