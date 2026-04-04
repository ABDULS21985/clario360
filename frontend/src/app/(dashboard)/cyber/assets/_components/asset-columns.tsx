'use client';

import { useRouter } from 'next/navigation';
import { ColumnDef, Row } from '@tanstack/react-table';
import {
  Server, Monitor, Cloud, Router, Wifi, AppWindow, Database, Box,
  MoreHorizontal, Pencil, Trash2, Tag, GitBranch,
} from 'lucide-react';
import Link from 'next/link';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { StatusBadge } from '@/components/shared/status-badge';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { timeAgo, cn } from '@/lib/utils';
import type { CyberAsset, AssetType, Criticality } from '@/types/cyber';

export const TYPE_ICONS: Record<AssetType, React.ElementType> = {
  server: Server,
  endpoint: Monitor,
  cloud_resource: Cloud,
  network_device: Router,
  iot_device: Wifi,
  application: AppWindow,
  database: Database,
  container: Box,
};

export const TYPE_LABELS: Record<AssetType, string> = {
  server: 'Server',
  endpoint: 'Endpoint',
  cloud_resource: 'Cloud',
  network_device: 'Network',
  iot_device: 'IoT',
  application: 'App',
  database: 'Database',
  container: 'Container',
};

const CRITICALITY_COLORS: Record<Criticality, string> = {
  critical: 'text-red-600',
  high: 'text-orange-600',
  medium: 'text-yellow-600',
  low: 'text-blue-600',
};

function vulnCountColor(count: number): string {
  if (count >= 20) return 'text-red-600 font-semibold';
  if (count >= 10) return 'text-orange-600 font-medium';
  if (count >= 1) return 'text-yellow-600';
  return 'text-green-600';
}

interface AssetActionsProps {
  asset: CyberAsset;
  onEdit?: (asset: CyberAsset) => void;
  onDelete?: (asset: CyberAsset) => void;
  onTag?: (asset: CyberAsset) => void;
  onRelationship?: (asset: CyberAsset) => void;
}

function AssetActions({ asset, onEdit, onDelete, onTag, onRelationship }: AssetActionsProps) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="sm" className="h-7 w-7 p-0">
          <MoreHorizontal className="h-4 w-4" />
          <span className="sr-only">Actions</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {onEdit && (
          <DropdownMenuItem onClick={() => onEdit(asset)}>
            <Pencil className="mr-2 h-3.5 w-3.5" /> Edit
          </DropdownMenuItem>
        )}
        {onTag && (
          <DropdownMenuItem onClick={() => onTag(asset)}>
            <Tag className="mr-2 h-3.5 w-3.5" /> Manage Tags
          </DropdownMenuItem>
        )}
        {onRelationship && (
          <DropdownMenuItem onClick={() => onRelationship(asset)}>
            <GitBranch className="mr-2 h-3.5 w-3.5" /> Add Relationship
          </DropdownMenuItem>
        )}
        {onDelete && (
          <DropdownMenuItem
            className="text-destructive"
            onClick={() => onDelete(asset)}
          >
            <Trash2 className="mr-2 h-3.5 w-3.5" /> Delete
          </DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

interface AssetColumnOptions {
  onEdit?: (asset: CyberAsset) => void;
  onDelete?: (asset: CyberAsset) => void;
  onTag?: (asset: CyberAsset) => void;
  onRelationship?: (asset: CyberAsset) => void;
}

export function getAssetColumns(options: AssetColumnOptions = {}): ColumnDef<CyberAsset>[] {
  return [
    {
      id: 'name',
      accessorKey: 'name',
      header: 'Name',
      cell: ({ row }: { row: Row<CyberAsset> }) => {
        const asset = row.original;
        const Icon = TYPE_ICONS[asset.type] ?? Server;
        return (
          <div className="flex items-center gap-2 min-w-0">
            <Icon className="h-4 w-4 shrink-0 text-muted-foreground" />
            <div className="min-w-0">
              <Link
                href={`/cyber/assets/${asset.id}`}
                className="font-medium hover:underline truncate block max-w-[120px] sm:max-w-[200px]"
              >
                {asset.name}
              </Link>
              {asset.hostname && (
                <p className="text-xs text-muted-foreground truncate max-w-[120px] sm:max-w-[200px]">{asset.hostname}</p>
              )}
            </div>
          </div>
        );
      },
      enableSorting: true,
    },
    {
      id: 'type',
      accessorKey: 'type',
      header: 'Type',
      cell: ({ row }: { row: Row<CyberAsset> }) => (
        <span className="text-sm">{TYPE_LABELS[row.original.type] ?? row.original.type}</span>
      ),
      enableSorting: true,
    },
    {
      id: 'ip_address',
      accessorKey: 'ip_address',
      header: 'IP Address',
      cell: ({ row }: { row: Row<CyberAsset> }) => (
        <span className="font-mono text-sm">{row.original.ip_address ?? '—'}</span>
      ),
    },
    {
      id: 'criticality',
      accessorKey: 'criticality',
      header: 'Criticality',
      cell: ({ row }: { row: Row<CyberAsset> }) => (
        <SeverityIndicator
          severity={row.original.criticality as 'critical' | 'high' | 'medium' | 'low'}
          showLabel
        />
      ),
      enableSorting: true,
    },
    {
      id: 'status',
      accessorKey: 'status',
      header: 'Status',
      cell: ({ row }: { row: Row<CyberAsset> }) => (
        <StatusBadge status={row.original.status} />
      ),
      enableSorting: true,
    },
    {
      id: 'vulnerability_count',
      accessorKey: 'vulnerability_count',
      header: 'Vulns',
      cell: ({ row }: { row: Row<CyberAsset> }) => {
        const count = row.original.vulnerability_count ?? 0;
        return (
          <span className={cn('text-sm tabular-nums', vulnCountColor(count))}>
            {count}
          </span>
        );
      },
      enableSorting: true,
    },
    {
      id: 'tags',
      header: 'Tags',
      cell: ({ row }: { row: Row<CyberAsset> }) => {
        const tags = row.original.tags ?? [];
        const visible = tags.slice(0, 3);
        const rest = tags.length - visible.length;
        return (
          <div className="flex flex-wrap gap-1">
            {visible.map((tag) => (
              <span key={tag} className="inline-flex rounded-full bg-muted px-1.5 py-0.5 text-xs">
                {tag}
              </span>
            ))}
            {rest > 0 && (
              <span className="inline-flex rounded-full bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
                +{rest} more
              </span>
            )}
          </div>
        );
      },
    },
    {
      id: 'last_seen_at',
      accessorKey: 'last_seen_at',
      header: 'Last Seen',
      cell: ({ row }: { row: Row<CyberAsset> }) => (
        <span className="text-sm text-muted-foreground">
          {row.original.last_seen_at ? timeAgo(row.original.last_seen_at) : '—'}
        </span>
      ),
      enableSorting: true,
    },
    {
      id: 'actions',
      header: '',
      cell: ({ row }: { row: Row<CyberAsset> }) => (
        <AssetActions asset={row.original} {...options} />
      ),
      enableSorting: false,
    },
  ];
}
