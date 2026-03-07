'use client';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';

export interface QueryAggregationRowState {
  id: string;
  column: string;
  func: string;
  alias: string;
}

interface QueryAggregationBuilderProps {
  rows: QueryAggregationRowState[];
  columnOptions: string[];
  onChange: (rows: QueryAggregationRowState[]) => void;
}

const AGGREGATIONS = ['count', 'sum', 'avg', 'min', 'max', 'count_distinct'] as const;

export function QueryAggregationBuilder({
  rows,
  columnOptions,
  onChange,
}: QueryAggregationBuilderProps) {
  return (
    <div className="space-y-3">
      {rows.map((row) => (
        <div key={row.id} className="grid gap-3 lg:grid-cols-[1fr_180px_1fr_auto]">
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

          <Select value={row.func} onValueChange={(value) => onChange(rows.map((item) => (item.id === row.id ? { ...item, func: value } : item)))}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {AGGREGATIONS.map((item) => (
                <SelectItem key={item} value={item}>
                  {item}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Input
            value={row.alias}
            onChange={(event) => onChange(rows.map((item) => (item.id === row.id ? { ...item, alias: event.target.value } : item)))}
            placeholder="Alias"
          />

          <Button type="button" variant="ghost" onClick={() => onChange(rows.filter((item) => item.id !== row.id))}>
            Remove
          </Button>
        </div>
      ))}
    </div>
  );
}
