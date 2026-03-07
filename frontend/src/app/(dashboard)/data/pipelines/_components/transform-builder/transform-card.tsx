'use client';

import { GripVertical, Trash2 } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import type { PipelineTransformDraft } from '@/app/(dashboard)/data/pipelines/_components/pipeline-wizard-types';
import { summarizeTransform, validateTransform } from '@/app/(dashboard)/data/pipelines/_components/pipeline-wizard-utils';
import { AggregateTransform } from '@/app/(dashboard)/data/pipelines/_components/transform-builder/aggregate-transform';
import { CastTransform } from '@/app/(dashboard)/data/pipelines/_components/transform-builder/cast-transform';
import { DedupTransform } from '@/app/(dashboard)/data/pipelines/_components/transform-builder/dedup-transform';
import { DeriveTransform } from '@/app/(dashboard)/data/pipelines/_components/transform-builder/derive-transform';
import { FilterTransform } from '@/app/(dashboard)/data/pipelines/_components/transform-builder/filter-transform';
import { MapTransform } from '@/app/(dashboard)/data/pipelines/_components/transform-builder/map-transform';
import { RenameTransform } from '@/app/(dashboard)/data/pipelines/_components/transform-builder/rename-transform';

interface TransformCardProps {
  value: PipelineTransformDraft;
  index: number;
  expanded: boolean;
  availableColumns: string[];
  onToggleExpand: () => void;
  onChange: (value: PipelineTransformDraft) => void;
  onRemove: () => void;
  onDragStart: () => void;
  onDragEnd: () => void;
  onDragOver: () => void;
  onDrop: () => void;
}

export function TransformCard({
  value,
  index,
  expanded,
  availableColumns,
  onToggleExpand,
  onChange,
  onRemove,
  onDragStart,
  onDragEnd,
  onDragOver,
  onDrop,
}: TransformCardProps) {
  const validationError = validateTransform(value);

  return (
    <div
      className="rounded-xl border bg-card"
      onDragOver={(event) => {
        event.preventDefault();
        onDragOver();
      }}
      onDrop={(event) => {
        event.preventDefault();
        onDrop();
      }}
    >
      <div className="flex items-center gap-3 border-b px-4 py-3">
        <button
          type="button"
          aria-label={`Drag transform ${index + 1}`}
          className="cursor-grab rounded-md border p-2 text-muted-foreground"
          draggable
          onDragStart={onDragStart}
          onDragEnd={onDragEnd}
        >
          <GripVertical className="h-4 w-4" />
        </button>

        <div className="flex-1">
          <div className="flex items-center gap-2">
            <Badge variant="outline" className="capitalize">
              {value.type.replace(/_/g, ' ')}
            </Badge>
            <span className="text-sm text-muted-foreground">Step {index + 1}</span>
          </div>
          <button type="button" className="mt-2 text-left text-sm font-medium" onClick={onToggleExpand}>
            {summarizeTransform(value)}
          </button>
          {validationError ? <div className="mt-1 text-xs text-destructive">{validationError}</div> : null}
        </div>

        <div className="flex items-center gap-2">
          <Button type="button" variant="ghost" size="sm" onClick={onToggleExpand}>
            {expanded ? 'Collapse' : 'Expand'}
          </Button>
          <Button type="button" variant="ghost" size="icon" onClick={onRemove} aria-label="Remove transform">
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {expanded ? (
        <div className="p-4">
          {value.type === 'rename' ? (
            <RenameTransform value={value} availableColumns={availableColumns} onChange={onChange} />
          ) : null}
          {value.type === 'cast' ? (
            <CastTransform value={value} availableColumns={availableColumns} onChange={onChange} />
          ) : null}
          {value.type === 'filter' ? (
            <FilterTransform value={value} availableColumns={availableColumns} onChange={onChange} />
          ) : null}
          {value.type === 'map_values' ? (
            <MapTransform value={value} availableColumns={availableColumns} onChange={onChange} />
          ) : null}
          {value.type === 'derive' ? (
            <DeriveTransform value={value} availableColumns={availableColumns} onChange={onChange} />
          ) : null}
          {value.type === 'deduplicate' ? (
            <DedupTransform value={value} availableColumns={availableColumns} onChange={onChange} />
          ) : null}
          {value.type === 'aggregate' ? (
            <AggregateTransform value={value} availableColumns={availableColumns} onChange={onChange} />
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

