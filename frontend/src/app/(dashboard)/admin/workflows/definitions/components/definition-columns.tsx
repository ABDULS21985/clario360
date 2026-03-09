'use client';

import type { ColumnDef } from '@tanstack/react-table';
import { Badge } from '@/components/ui/badge';
import {
  dateColumn,
  statusColumn,
  actionsColumn,
} from '@/components/shared/data-table/columns/common-columns';
import { workflowDefinitionStatusConfig } from '@/lib/status-configs';
import {
  Calendar,
  Globe,
  MousePointerClick,
  Webhook,
} from 'lucide-react';
import type { WorkflowDefinition } from '@/types/models';
import { titleCase } from '@/lib/format';

const triggerIcons: Record<string, React.ElementType> = {
  manual: MousePointerClick,
  event: Globe,
  schedule: Calendar,
  webhook: Webhook,
};

interface DefinitionColumnOptions {
  onEdit: (def: WorkflowDefinition) => void;
  onView: (def: WorkflowDefinition) => void;
  onPublish: (def: WorkflowDefinition) => void;
  onArchive: (def: WorkflowDefinition) => void;
  onClone: (def: WorkflowDefinition) => void;
  onDelete: (def: WorkflowDefinition) => void;
}

export function getDefinitionColumns(
  options: DefinitionColumnOptions,
): ColumnDef<WorkflowDefinition>[] {
  const { onEdit, onView, onPublish, onArchive, onClone, onDelete } = options;

  return [
    {
      id: 'name',
      accessorKey: 'name',
      header: 'Name',
      cell: ({ getValue, row }) => {
        const name = getValue() as string;
        return (
          <button
            onClick={() => onView(row.original)}
            className="text-sm font-medium text-left hover:underline"
          >
            {name}
          </button>
        );
      },
      enableSorting: true,
    },
    {
      id: 'category',
      accessorKey: 'category',
      header: 'Category',
      cell: ({ getValue }) => {
        const category = getValue() as string;
        return (
          <Badge variant="secondary" className="text-xs">
            {titleCase(category)}
          </Badge>
        );
      },
      enableSorting: true,
    },
    statusColumn<WorkflowDefinition>(
      'status',
      'Status',
      workflowDefinitionStatusConfig,
    ),
    {
      id: 'version',
      accessorKey: 'version',
      header: 'Version',
      cell: ({ getValue }) => (
        <span className="text-sm text-muted-foreground">v{getValue() as number}</span>
      ),
      enableSorting: true,
      size: 80,
    },
    {
      id: 'trigger',
      header: 'Trigger',
      cell: ({ row }) => {
        const trigger = row.original.trigger;
        const Icon = triggerIcons[trigger.type] ?? Globe;
        return (
          <div className="flex items-center gap-1.5">
            <Icon className="h-3.5 w-3.5 text-muted-foreground" />
            <span className="text-sm">{titleCase(trigger.type)}</span>
          </div>
        );
      },
      size: 120,
      enableSorting: false,
    },
    {
      id: 'steps',
      header: 'Steps',
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground">
          {row.original.steps.length}
        </span>
      ),
      size: 70,
      enableSorting: false,
    },
    {
      id: 'instance_count',
      accessorKey: 'instance_count',
      header: 'Instances',
      cell: ({ getValue }) => (
        <span className="text-sm text-muted-foreground">
          {getValue() as number}
        </span>
      ),
      size: 90,
      enableSorting: true,
    },
    dateColumn<WorkflowDefinition>('updated_at', 'Last Updated', {
      relative: true,
    }),
    actionsColumn<WorkflowDefinition>((def) => [
      ...(def.status === 'draft'
        ? [{ label: 'Edit', onClick: () => onEdit(def) }]
        : []),
      { label: 'View', onClick: () => onView(def) },
      ...(def.status === 'draft'
        ? [{ label: 'Publish', onClick: () => onPublish(def) }]
        : []),
      ...(def.status === 'active'
        ? [{ label: 'Archive', onClick: () => onArchive(def) }]
        : []),
      { label: 'Clone', onClick: () => onClone(def) },
      ...(def.status === 'draft'
        ? [
            {
              label: 'Delete',
              onClick: () => onDelete(def),
              variant: 'destructive' as const,
            },
          ]
        : []),
    ]),
  ];
}
