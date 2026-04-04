'use client';

import { useCallback } from 'react';
import { Plus, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { WorkflowCondition } from '@/types/models';

const OPERATORS: { value: WorkflowCondition['operator']; label: string }[] = [
  { value: 'eq', label: '=' },
  { value: 'neq', label: '!=' },
  { value: 'gt', label: '>' },
  { value: 'gte', label: '>=' },
  { value: 'lt', label: '<' },
  { value: 'lte', label: '<=' },
  { value: 'in', label: 'in' },
  { value: 'not_in', label: 'not in' },
  { value: 'contains', label: 'contains' },
  { value: 'matches', label: 'matches' },
];

interface ConditionBuilderProps {
  conditions: WorkflowCondition[];
  onChange: (conditions: WorkflowCondition[]) => void;
  readOnly?: boolean;
}

export function ConditionBuilder({ conditions, onChange, readOnly }: ConditionBuilderProps) {
  const handleAdd = useCallback(() => {
    onChange([
      ...conditions,
      { field: '', operator: 'eq', value: '', logic: 'and' },
    ]);
  }, [conditions, onChange]);

  const handleRemove = useCallback(
    (index: number) => {
      onChange(conditions.filter((_, i) => i !== index));
    },
    [conditions, onChange],
  );

  const handleUpdate = useCallback(
    (index: number, updates: Partial<WorkflowCondition>) => {
      onChange(
        conditions.map((c, i) => (i === index ? { ...c, ...updates } : c)),
      );
    },
    [conditions, onChange],
  );

  return (
    <div className="space-y-2">
      <Label className="text-xs">Conditions</Label>

      {conditions.length === 0 && (
        <p className="text-xs text-muted-foreground">
          No conditions defined. All paths are taken.
        </p>
      )}

      {conditions.map((cond, i) => (
        <div key={i} className="space-y-1.5 rounded-md border bg-muted/30 p-2">
          {i > 0 && (
            <Select
              value={cond.logic ?? 'and'}
              onValueChange={(v) =>
                handleUpdate(i, { logic: v as 'and' | 'or' })
              }
              disabled={readOnly}
            >
              <SelectTrigger className="h-6 w-16 text-[10px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="and">AND</SelectItem>
                <SelectItem value="or">OR</SelectItem>
              </SelectContent>
            </Select>
          )}

          <div className="flex items-center gap-1.5">
            {/* Field */}
            <Input
              value={cond.field}
              onChange={(e) => handleUpdate(i, { field: e.target.value })}
              placeholder="variables.field"
              disabled={readOnly}
              className="h-7 text-xs flex-1"
            />

            {/* Operator */}
            <Select
              value={cond.operator}
              onValueChange={(v) =>
                handleUpdate(i, { operator: v as WorkflowCondition['operator'] })
              }
              disabled={readOnly}
            >
              <SelectTrigger className="h-7 w-20 text-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {OPERATORS.map((op) => (
                  <SelectItem key={op.value} value={op.value}>
                    {op.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            {/* Value */}
            <Input
              value={cond.value !== undefined ? String(cond.value) : ''}
              onChange={(e) => handleUpdate(i, { value: e.target.value })}
              placeholder="value"
              disabled={readOnly}
              className="h-7 text-xs flex-1"
            />

            {/* Remove */}
            {!readOnly && (
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7 shrink-0"
                onClick={() => handleRemove(i)}
              >
                <Trash2 className="h-3 w-3" />
              </Button>
            )}
          </div>
        </div>
      ))}

      {!readOnly && (
        <Button
          variant="outline"
          size="sm"
          className="w-full h-7 text-xs"
          onClick={handleAdd}
        >
          <Plus className="mr-1 h-3 w-3" />
          Add Condition
        </Button>
      )}
    </div>
  );
}
