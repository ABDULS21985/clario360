'use client';

import { Badge } from '@/components/ui/badge';
import type { UebaFrequencyEntry } from './types';

export function TableAccessList({
  expectedTables,
  actualTables,
}: {
  expectedTables: UebaFrequencyEntry[];
  actualTables: string[];
}) {
  const expected = new Set(expectedTables.map((item) => item.name.toLowerCase()));

  return (
    <div className="space-y-2">
      {actualTables.map((table) => {
        const known = expected.has(table.toLowerCase());
        return (
          <div key={table} className="flex items-center justify-between rounded-lg border p-3">
            <span className="font-mono text-xs">{table}</span>
            <Badge variant={known ? 'outline' : 'warning'}>
              {known ? 'known' : 'new'}
            </Badge>
          </div>
        );
      })}
      {actualTables.length === 0 && <p className="text-sm text-muted-foreground">No recent table access found.</p>}
    </div>
  );
}
