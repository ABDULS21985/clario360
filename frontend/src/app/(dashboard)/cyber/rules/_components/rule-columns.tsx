'use client';

import { ColumnDef, Row } from '@tanstack/react-table';
import { Switch } from '@/components/ui/switch';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { MoreHorizontal, Pencil, FlaskConical } from 'lucide-react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { timeAgo } from '@/lib/utils';
import type { DetectionRule } from '@/types/cyber';

const RULE_TYPE_COLORS: Record<string, string> = {
  sigma: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300',
  threshold: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300',
  correlation: 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-300',
  anomaly: 'bg-teal-100 text-teal-800 dark:bg-teal-900/30 dark:text-teal-300',
};

interface RuleColumnOptions {
  onToggle?: (rule: DetectionRule) => void;
  onEdit?: (rule: DetectionRule) => void;
  onTest?: (rule: DetectionRule) => void;
}

export function getRuleColumns(options: RuleColumnOptions = {}): ColumnDef<DetectionRule>[] {
  return [
    {
      id: 'enabled',
      header: 'Active',
      cell: ({ row }: { row: Row<DetectionRule> }) => (
        <Switch
          checked={row.original.enabled}
          onCheckedChange={() => options.onToggle?.(row.original)}
          aria-label={`Toggle ${row.original.name}`}
        />
      ),
    },
    {
      id: 'name',
      accessorKey: 'name',
      header: 'Rule',
      cell: ({ row }: { row: Row<DetectionRule> }) => {
        const rule = row.original;
        return (
          <div>
            <p className="font-medium">{rule.name}</p>
            <p className="text-xs text-muted-foreground line-clamp-1 max-w-xs">{rule.description}</p>
          </div>
        );
      },
      enableSorting: true,
    },
    {
      id: 'type',
      accessorKey: 'type',
      header: 'Type',
      cell: ({ row }: { row: Row<DetectionRule> }) => (
        <span className={`rounded-full px-2 py-0.5 text-xs font-medium capitalize ${RULE_TYPE_COLORS[row.original.type] ?? ''}`}>
          {row.original.type}
        </span>
      ),
      enableSorting: true,
    },
    {
      id: 'severity',
      accessorKey: 'severity',
      header: 'Severity',
      cell: ({ row }: { row: Row<DetectionRule> }) => (
        <SeverityIndicator severity={row.original.severity} showLabel />
      ),
      enableSorting: true,
    },
    {
      id: 'trigger_count',
      accessorKey: 'trigger_count',
      header: 'Triggers',
      cell: ({ row }: { row: Row<DetectionRule> }) => (
        <span className="tabular-nums text-sm">{row.original.trigger_count.toLocaleString()}</span>
      ),
      enableSorting: true,
    },
    {
      id: 'false_positive_rate',
      accessorKey: 'false_positive_rate',
      header: 'FP Rate',
      cell: ({ row }: { row: Row<DetectionRule> }) => {
        const rate = row.original.false_positive_rate;
        const pct = (rate * 100).toFixed(1);
        return (
          <span className={`text-sm ${parseFloat(pct) > 10 ? 'text-orange-600 font-medium' : 'text-muted-foreground'}`}>
            {pct}%
          </span>
        );
      },
      enableSorting: true,
    },
    {
      id: 'mitre',
      header: 'MITRE',
      cell: ({ row }: { row: Row<DetectionRule> }) => {
        const ids = row.original.mitre_technique_ids ?? [];
        if (ids.length === 0) return <span className="text-muted-foreground">—</span>;
        return (
          <div className="flex gap-1 flex-wrap">
            {ids.slice(0, 2).map((id) => (
              <Badge key={id} variant="outline" className="font-mono text-xs">{id}</Badge>
            ))}
            {ids.length > 2 && <span className="text-xs text-muted-foreground">+{ids.length - 2}</span>}
          </div>
        );
      },
    },
    {
      id: 'last_triggered',
      accessorKey: 'last_triggered',
      header: 'Last Triggered',
      cell: ({ row }: { row: Row<DetectionRule> }) => (
        <span className="text-sm text-muted-foreground">
          {row.original.last_triggered ? timeAgo(row.original.last_triggered) : '—'}
        </span>
      ),
      enableSorting: true,
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }: { row: Row<DetectionRule> }) => (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm" className="h-7 w-7 p-0">
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => options.onEdit?.(row.original)}>
              <Pencil className="mr-2 h-3.5 w-3.5" /> Edit
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => options.onTest?.(row.original)}>
              <FlaskConical className="mr-2 h-3.5 w-3.5" /> Test Rule
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      ),
      enableSorting: false,
    },
  ];
}
