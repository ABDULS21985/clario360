'use client';

import { useState } from 'react';
import { ColumnDef, Row } from '@tanstack/react-table';
import { Switch } from '@/components/ui/switch';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  MoreHorizontal,
  Pencil,
  FlaskConical,
  Copy,
  Trash2,
  Bell,
  AlertTriangle,
} from 'lucide-react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { RulePerformanceCard } from './rule-performance-card';
import { timeAgo } from '@/lib/utils';
import type { DetectionRule } from '@/types/cyber';

const RULE_TYPE_COLORS: Record<string, string> = {
  sigma: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300',
  threshold: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300',
  correlation: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300',
  anomaly: 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-300',
};

interface RuleColumnOptions {
  onToggle?: (rule: DetectionRule) => void;
  onEdit?: (rule: DetectionRule) => void;
  onTest?: (rule: DetectionRule) => void;
  onDuplicate?: (rule: DetectionRule) => void;
  onDelete?: (rule: DetectionRule) => void;
  onViewAlerts?: (rule: DetectionRule) => void;
}

function RuleToggleCell({
  rule,
  onToggle,
}: {
  rule: DetectionRule;
  onToggle?: (rule: DetectionRule) => void;
}) {
  const [optimisticEnabled, setOptimisticEnabled] = useState(rule.enabled);
  const fpHigh = rule.false_positive_rate * 100 > 50;

  const handleChange = () => {
    setOptimisticEnabled((prev) => !prev);
    onToggle?.(rule);
  };

  return (
    <div className="flex items-center gap-1.5">
      <Switch
        checked={optimisticEnabled}
        onCheckedChange={handleChange}
        aria-label={`Toggle ${rule.name}`}
      />
      {fpHigh && !optimisticEnabled && (
        <span title="High FP rate — auto-disable risk">
          <AlertTriangle className="h-3.5 w-3.5 text-red-500" aria-hidden />
        </span>
      )}
    </div>
  );
}

function ActionsCell({
  rule,
  onEdit,
  onTest,
  onDuplicate,
  onDelete,
  onViewAlerts,
}: {
  rule: DetectionRule;
  onEdit?: (rule: DetectionRule) => void;
  onTest?: (rule: DetectionRule) => void;
  onDuplicate?: (rule: DetectionRule) => void;
  onDelete?: (rule: DetectionRule) => void;
  onViewAlerts?: (rule: DetectionRule) => void;
}) {
  const [deleteOpen, setDeleteOpen] = useState(false);

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="sm" className="h-7 w-7 p-0">
            <MoreHorizontal className="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem onClick={() => onEdit?.(rule)}>
            <Pencil className="mr-2 h-3.5 w-3.5" />
            {rule.is_template ? 'Customize' : 'Edit'}
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => onTest?.(rule)}>
            <FlaskConical className="mr-2 h-3.5 w-3.5" /> Test Rule
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => onDuplicate?.(rule)}>
            <Copy className="mr-2 h-3.5 w-3.5" /> Duplicate
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => onViewAlerts?.(rule)}>
            <Bell className="mr-2 h-3.5 w-3.5" /> View Alerts
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            onClick={() => setDeleteOpen(true)}
            className="text-destructive focus:text-destructive"
          >
            <Trash2 className="mr-2 h-3.5 w-3.5" /> Delete
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title="Delete Detection Rule"
        description={`Are you sure you want to delete "${rule.name}"? This action cannot be undone.`}
        confirmLabel="Delete"
        variant="destructive"
        onConfirm={() => {
          setDeleteOpen(false);
          onDelete?.(rule);
        }}
      />
    </>
  );
}

export function getRuleColumns(options: RuleColumnOptions = {}): ColumnDef<DetectionRule>[] {
  return [
    {
      id: 'enabled',
      header: 'Active',
      cell: ({ row }: { row: Row<DetectionRule> }) => (
        <RuleToggleCell rule={row.original} onToggle={options.onToggle} />
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
            <p className="line-clamp-1 max-w-xs text-xs text-muted-foreground">{rule.description}</p>
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
        <span
          className={`rounded-full px-2 py-0.5 text-xs font-medium capitalize ${RULE_TYPE_COLORS[row.original.type] ?? ''}`}
        >
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
      id: 'mitre',
      header: 'MITRE',
      cell: ({ row }: { row: Row<DetectionRule> }) => {
        const ids = row.original.mitre_technique_ids ?? [];
        if (ids.length === 0) return <span className="text-muted-foreground">—</span>;
        return (
          <div className="flex flex-wrap gap-1">
            {ids.slice(0, 2).map((id) => (
              <Badge key={id} variant="outline" className="font-mono text-xs">
                {id}
              </Badge>
            ))}
            {ids.length > 2 && (
              <span className="text-xs text-muted-foreground">+{ids.length - 2}</span>
            )}
          </div>
        );
      },
    },
    {
      id: 'performance',
      header: 'Performance',
      cell: ({ row }: { row: Row<DetectionRule> }) => (
        <RulePerformanceCard rule={row.original} />
      ),
    },
    {
      id: 'last_triggered',
      accessorKey: 'last_triggered',
      header: 'Last Triggered',
      cell: ({ row }: { row: Row<DetectionRule> }) => (
        <span className="text-sm text-muted-foreground">
          {row.original.last_triggered ? timeAgo(row.original.last_triggered) : 'Never'}
        </span>
      ),
      enableSorting: true,
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }: { row: Row<DetectionRule> }) => (
        <ActionsCell
          rule={row.original}
          onEdit={options.onEdit}
          onTest={options.onTest}
          onDuplicate={options.onDuplicate}
          onDelete={options.onDelete}
          onViewAlerts={options.onViewAlerts}
        />
      ),
      enableSorting: false,
    },
  ];
}
