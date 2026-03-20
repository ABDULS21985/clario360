'use client';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';

export interface QueryFilterRowState {
  id: string;
  column: string;
  operator: string;
  value: string;
  secondaryValue: string;
}

interface QueryFilterBuilderProps {
  rows: QueryFilterRowState[];
  columnOptions: string[];
  onChange: (rows: QueryFilterRowState[]) => void;
}

const OPERATORS = [
  { label: 'Equals', value: 'eq' },
  { label: 'Not equals', value: 'neq' },
  { label: 'Greater than', value: 'gt' },
  { label: 'Greater than or equals', value: 'gte' },
  { label: 'Less than', value: 'lt' },
  { label: 'Less than or equals', value: 'lte' },
  { label: 'In', value: 'in' },
  { label: 'Not in', value: 'not_in' },
  { label: 'Like', value: 'like' },
  { label: 'ILike', value: 'ilike' },
  { label: 'Between', value: 'between' },
  { label: 'Is null', value: 'is_null' },
  { label: 'Is not null', value: 'is_not_null' },
] as const;

export function QueryFilterBuilder({
  rows,
  columnOptions,
  onChange,
}: QueryFilterBuilderProps) {
  return (
    <div className="space-y-3">
      {rows.map((row) => (
        <div key={row.id} className="grid grid-cols-1 gap-3 lg:grid-cols-[1fr_180px_1fr_auto]">
          <Select value={row.column} onValueChange={(value) => onChange(rows.map((item) => (item.id === row.id ? { ...item, column: value } : item)))}>
            <SelectTrigger>
              <SelectValue placeholder="Column" />
            </SelectTrigger>
            <SelectContent>
              {columnOptions.map((column) => (
                <SelectItem key={column} value={column}>
                  {column}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Select value={row.operator} onValueChange={(value) => onChange(rows.map((item) => (item.id === row.id ? { ...item, operator: value } : item)))}>
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

          {row.operator === 'is_null' || row.operator === 'is_not_null' ? (
            <div className="flex items-center rounded-md border px-3 text-sm text-muted-foreground">No value required</div>
          ) : row.operator === 'between' ? (
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
              <Input
                value={row.value}
                onChange={(event) => onChange(rows.map((item) => (item.id === row.id ? { ...item, value: event.target.value } : item)))}
                placeholder="From"
              />
              <Input
                value={row.secondaryValue}
                onChange={(event) => onChange(rows.map((item) => (item.id === row.id ? { ...item, secondaryValue: event.target.value } : item)))}
                placeholder="To"
              />
            </div>
          ) : (
            <Input
              value={row.value}
              onChange={(event) => onChange(rows.map((item) => (item.id === row.id ? { ...item, value: event.target.value } : item)))}
              placeholder={row.operator === 'in' || row.operator === 'not_in' ? 'Comma-separated values' : 'Value'}
            />
          )}

          <Button type="button" variant="ghost" onClick={() => onChange(rows.filter((item) => item.id !== row.id))}>
            Remove
          </Button>
        </div>
      ))}
    </div>
  );
}
