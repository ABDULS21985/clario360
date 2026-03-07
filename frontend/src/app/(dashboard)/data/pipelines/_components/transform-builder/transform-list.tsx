'use client';

import { useEffect, useMemo, useState } from 'react';
import { Plus, Sparkles } from 'lucide-react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Button } from '@/components/ui/button';
import type { JsonValue } from '@/lib/data-suite';
import type { PipelineTransformDraft } from '@/app/(dashboard)/data/pipelines/_components/pipeline-wizard-types';
import { createEmptyTransform } from '@/app/(dashboard)/data/pipelines/_components/pipeline-wizard-utils';
import { TransformCard } from '@/app/(dashboard)/data/pipelines/_components/transform-builder/transform-card';

interface TransformListProps {
  transforms: PipelineTransformDraft[];
  availableColumns: string[];
  previewBeforeRows: Array<Record<string, JsonValue>>;
  previewAfterRows: Array<Record<string, JsonValue>>;
  previewError: string | null;
  onChange: (value: PipelineTransformDraft[]) => void;
  onPreview: () => void;
}

const TRANSFORM_TYPES: Array<{ label: string; value: PipelineTransformDraft['type'] }> = [
  { label: 'Rename', value: 'rename' },
  { label: 'Cast', value: 'cast' },
  { label: 'Filter', value: 'filter' },
  { label: 'Map Values', value: 'map_values' },
  { label: 'Derive', value: 'derive' },
  { label: 'Deduplicate', value: 'deduplicate' },
  { label: 'Aggregate', value: 'aggregate' },
];

export function TransformList({
  transforms,
  availableColumns,
  previewBeforeRows,
  previewAfterRows,
  previewError,
  onChange,
  onPreview,
}: TransformListProps) {
  const [draggedId, setDraggedId] = useState<string | null>(null);
  const [expandedIds, setExpandedIds] = useState<string[]>([]);

  useEffect(() => {
    setExpandedIds((current) => Array.from(new Set([...current, ...transforms.map((transform) => transform.id)])));
  }, [transforms]);

  const previewColumns = useMemo(() => {
    const columns = new Set<string>();
    [...previewBeforeRows, ...previewAfterRows].forEach((row) => {
      Object.keys(row).forEach((key) => columns.add(key));
    });
    return Array.from(columns);
  }, [previewAfterRows, previewBeforeRows]);

  const reorderTo = (targetId: string) => {
    if (!draggedId || draggedId === targetId) {
      return;
    }
    const next = [...transforms];
    const draggedIndex = next.findIndex((transform) => transform.id === draggedId);
    const targetIndex = next.findIndex((transform) => transform.id === targetId);
    if (draggedIndex < 0 || targetIndex < 0) {
      return;
    }
    const [dragged] = next.splice(draggedIndex, 1);
    next.splice(targetIndex, 0, dragged);
    onChange(next);
  };

  return (
    <div className="space-y-4">
      <div className="space-y-3">
        {transforms.length === 0 ? (
          <div className="rounded-xl border border-dashed p-6 text-sm text-muted-foreground">
            Add one or more transformations to define the pipeline flow.
          </div>
        ) : (
          transforms.map((transform, index) => (
            <TransformCard
              key={transform.id}
              value={transform}
              index={index}
              expanded={expandedIds.includes(transform.id)}
              availableColumns={availableColumns}
              onToggleExpand={() =>
                setExpandedIds((current) =>
                  current.includes(transform.id)
                    ? current.filter((item) => item !== transform.id)
                    : [...current, transform.id],
                )
              }
              onChange={(next) =>
                onChange(transforms.map((item) => (item.id === transform.id ? next : item)))
              }
              onRemove={() => onChange(transforms.filter((item) => item.id !== transform.id))}
              onDragStart={() => setDraggedId(transform.id)}
              onDragEnd={() => setDraggedId(null)}
              onDragOver={() => undefined}
              onDrop={() => {
                reorderTo(transform.id);
                setDraggedId(null);
              }}
            />
          ))
        )}
      </div>

      <div className="flex flex-wrap items-center gap-3">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button type="button" variant="outline">
              <Plus className="mr-2 h-4 w-4" />
              Add transformation
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start">
            {TRANSFORM_TYPES.map((type) => (
              <DropdownMenuItem
                key={type.value}
                onClick={() => onChange([...transforms, createEmptyTransform(type.value)])}
              >
                {type.label}
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>

        <Button type="button" onClick={onPreview} disabled={availableColumns.length === 0}>
          <Sparkles className="mr-2 h-4 w-4" />
          Preview Transformation (first 5 rows)
        </Button>
      </div>

      {previewError ? <div className="rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">{previewError}</div> : null}

      {previewColumns.length > 0 && !previewError ? (
        <div className="grid gap-4 xl:grid-cols-2">
          <PreviewTable title="Before" columns={previewColumns} rows={previewBeforeRows} />
          <PreviewTable title="After" columns={previewColumns} rows={previewAfterRows} />
        </div>
      ) : null}
    </div>
  );
}

function PreviewTable({
  title,
  columns,
  rows,
}: {
  title: string;
  columns: string[];
  rows: Array<Record<string, JsonValue>>;
}) {
  return (
    <div className="rounded-xl border">
      <div className="border-b px-4 py-3">
        <div className="font-medium">{title}</div>
      </div>
      <div className="overflow-x-auto">
        <table className="min-w-full text-sm">
          <thead>
            <tr className="border-b bg-muted/20 text-left">
              {columns.map((column) => (
                <th key={column} className="px-3 py-2 font-medium">
                  {column}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((row, rowIndex) => (
              <tr key={`${title}-${rowIndex}`} className="border-b">
                {columns.map((column) => (
                  <td key={`${title}-${rowIndex}-${column}`} className="px-3 py-2 text-muted-foreground">
                    {`${row[column] ?? '—'}`}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

