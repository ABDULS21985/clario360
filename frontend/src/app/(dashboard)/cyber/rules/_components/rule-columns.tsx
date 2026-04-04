'use client';

import Link from 'next/link';
import { useState } from 'react';
import type { ColumnDef, Row } from '@tanstack/react-table';
import { Copy, Eye, FlaskConical, MoreHorizontal, Pencil, Trash2 } from 'lucide-react';

import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { Switch } from '@/components/ui/switch';
import { getRuleTypeColor, getRuleTypeLabel } from '@/lib/cyber-rules';
import { timeAgo } from '@/lib/utils';
import type { DetectionRule } from '@/types/cyber';

import { RulePerformanceCard } from './rule-performance-card';

interface RuleColumnOptions {
  onToggle: (rule: DetectionRule) => void;
  onEdit: (rule: DetectionRule) => void;
  onDuplicate: (rule: DetectionRule) => void;
  onDelete: (rule: DetectionRule) => void;
  onTest: (rule: DetectionRule) => void;
}

function ActionsCell({
  rule,
  options,
}: {
  rule: DetectionRule;
  options: RuleColumnOptions;
}) {
  const [deleteOpen, setDeleteOpen] = useState(false);

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="icon" className="h-8 w-8">
            <MoreHorizontal className="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem asChild>
            <Link href={`/cyber/detection-rules/${rule.id}`}>
              <Eye className="mr-2 h-4 w-4" />
              View Details
            </Link>
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => options.onEdit(rule)}>
            <Pencil className="mr-2 h-4 w-4" />
            Edit
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => options.onDuplicate(rule)}>
            <Copy className="mr-2 h-4 w-4" />
            Duplicate
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => options.onTest(rule)}>
            <FlaskConical className="mr-2 h-4 w-4" />
            Test Rule
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem className="text-red-600 focus:text-red-600" onClick={() => setDeleteOpen(true)}>
            <Trash2 className="mr-2 h-4 w-4" />
            Delete
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title="Delete detection rule"
        description={`Delete "${rule.name}"? The rule will be soft-deleted and no longer available in the detection engine.`}
        confirmLabel="Delete"
        variant="destructive"
        onConfirm={() => options.onDelete(rule)}
      />
    </>
  );
}

export function getRuleColumns(options: RuleColumnOptions): ColumnDef<DetectionRule>[] {
  return [
    {
      id: 'name',
      accessorKey: 'name',
      header: 'Rule Name',
      enableSorting: true,
      cell: ({ row }: { row: Row<DetectionRule> }) => (
        <div className="space-y-1">
          <Link href={`/cyber/detection-rules/${row.original.id}`} className="font-medium text-slate-950 hover:text-emerald-700 hover:underline">
            {row.original.name}
          </Link>
          <p className="max-w-md truncate text-xs text-muted-foreground">{row.original.description || 'No description provided.'}</p>
        </div>
      ),
    },
    {
      id: 'type',
      accessorKey: 'rule_type',
      header: 'Type',
      cell: ({ row }: { row: Row<DetectionRule> }) => (
        <Badge className={getRuleTypeColor(row.original.rule_type)}>
          {getRuleTypeLabel(row.original.rule_type)}
        </Badge>
      ),
    },
    {
      id: 'severity',
      accessorKey: 'severity',
      header: 'Severity',
      cell: ({ row }: { row: Row<DetectionRule> }) => (
        <SeverityIndicator severity={row.original.severity} showLabel />
      ),
    },
    {
      id: 'mitre',
      header: 'MITRE Technique',
      cell: ({ row }: { row: Row<DetectionRule> }) => {
        const techniques = row.original.mitre_technique_ids ?? [];
        if (techniques.length === 0) {
          return <span className="text-sm text-muted-foreground">Unmapped</span>;
        }
        return (
          <div className="flex flex-wrap gap-1">
            {techniques.slice(0, 2).map((techniqueId) => (
              <Badge key={techniqueId} variant="outline" className="font-mono text-xs">
                {techniqueId}
              </Badge>
            ))}
            {techniques.length > 2 ? (
              <span className="text-xs text-muted-foreground">+{techniques.length - 2}</span>
            ) : null}
          </div>
        );
      },
    },
    {
      id: 'enabled',
      accessorKey: 'enabled',
      header: 'Status',
      cell: ({ row }: { row: Row<DetectionRule> }) => (
        <div className="flex items-center gap-3">
          <Switch checked={row.original.enabled} onCheckedChange={() => options.onToggle(row.original)} />
          <span className="text-sm text-muted-foreground">{row.original.enabled ? 'Enabled' : 'Disabled'}</span>
        </div>
      ),
    },
    {
      id: 'performance',
      header: 'TP / FP',
      cell: ({ row }: { row: Row<DetectionRule> }) => <RulePerformanceCard rule={row.original} />,
    },
    {
      id: 'trigger_count',
      accessorKey: 'trigger_count',
      header: 'Alerts Generated',
      enableSorting: true,
      cell: ({ row }: { row: Row<DetectionRule> }) => (
        <span className="tabular-nums text-sm">{row.original.trigger_count.toLocaleString()}</span>
      ),
    },
    {
      id: 'last_triggered_at',
      accessorKey: 'last_triggered_at',
      header: 'Last Triggered',
      enableSorting: true,
      cell: ({ row }: { row: Row<DetectionRule> }) => (
        <span className="text-sm text-muted-foreground">
          {row.original.last_triggered_at ? timeAgo(row.original.last_triggered_at) : 'Never'}
        </span>
      ),
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }: { row: Row<DetectionRule> }) => <ActionsCell rule={row.original} options={options} />,
    },
  ];
}
