'use client';

import { Button } from '@/components/ui/button';
import { type SavedQuery } from '@/lib/data-suite';
import { formatMaybeDateTime } from '@/lib/data-suite/utils';

interface SavedQueriesListProps {
  queries: SavedQuery[];
  modelNames: Record<string, string>;
  onRun: (query: SavedQuery) => void;
  onEdit: (query: SavedQuery) => void;
  onDelete: (query: SavedQuery) => void;
}

export function SavedQueriesList({
  queries,
  modelNames,
  onRun,
  onEdit,
  onDelete,
}: SavedQueriesListProps) {
  if (queries.length === 0) {
    return <p className="text-sm text-muted-foreground">No saved queries yet.</p>;
  }

  return (
    <div className="rounded-lg border overflow-x-auto">
      <table className="min-w-full text-sm">
        <thead>
          <tr className="border-b text-left">
            <th className="px-3 py-2 font-medium">Name</th>
            <th className="px-3 py-2 font-medium">Model</th>
            <th className="px-3 py-2 font-medium">Last Run</th>
            <th className="px-3 py-2 font-medium">Runs</th>
            <th className="px-3 py-2 font-medium"></th>
          </tr>
        </thead>
        <tbody>
          {queries.map((query) => (
            <tr key={query.id} className="border-b">
              <td className="px-3 py-2">
                <div className="font-medium">{query.name}</div>
                <div className="text-xs text-muted-foreground">{query.visibility}</div>
              </td>
              <td className="px-3 py-2 text-muted-foreground">{modelNames[query.model_id] ?? query.model_id}</td>
              <td className="px-3 py-2 text-muted-foreground">{formatMaybeDateTime(query.last_run_at)}</td>
              <td className="px-3 py-2 text-muted-foreground">{query.run_count}</td>
              <td className="px-3 py-2">
                <div className="flex gap-2">
                  <Button type="button" size="sm" variant="outline" onClick={() => onRun(query)}>
                    Run
                  </Button>
                  <Button type="button" size="sm" variant="outline" onClick={() => onEdit(query)}>
                    Edit
                  </Button>
                  <Button type="button" size="sm" variant="ghost" onClick={() => onDelete(query)}>
                    Delete
                  </Button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
