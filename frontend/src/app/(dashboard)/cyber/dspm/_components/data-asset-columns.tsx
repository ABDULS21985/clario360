'use client';

import { ColumnDef, Row } from '@tanstack/react-table';
import { Badge } from '@/components/ui/badge';
import { CheckCircle, XCircle, Globe, Lock } from 'lucide-react';
import type { DataAsset } from '@/types/cyber';

const CLASSIFICATION_COLORS: Record<string, string> = {
  public: 'bg-green-100 text-green-700',
  internal: 'bg-blue-100 text-blue-700',
  confidential: 'bg-amber-100 text-amber-800',
  restricted: 'bg-red-100 text-red-700',
  top_secret: 'bg-purple-100 text-purple-700',
};

function ScoreBar({ score, invert = false }: { score: number; invert?: boolean }) {
  const color = invert
    ? score <= 30 ? 'bg-green-500' : score <= 60 ? 'bg-amber-500' : 'bg-red-500'
    : score >= 80 ? 'bg-green-500' : score >= 60 ? 'bg-amber-500' : 'bg-red-500';
  return (
    <div className="flex items-center gap-2">
      <div className="h-1.5 w-16 overflow-hidden rounded-full bg-muted">
        <div className={`h-full rounded-full ${color} transition-all`} style={{ width: `${score}%` }} />
      </div>
      <span className="text-xs tabular-nums">{score.toFixed(0)}</span>
    </div>
  );
}

export const dataAssetColumns: ColumnDef<DataAsset>[] = [
  {
    id: 'asset_name',
    accessorKey: 'asset_name',
    header: 'Asset',
    cell: ({ row }: { row: Row<DataAsset> }) => {
      const asset = row.original;
      return (
        <div>
          <p className="font-medium text-sm">{asset.asset_name}</p>
          <p className="text-xs capitalize text-muted-foreground">{asset.asset_type}</p>
        </div>
      );
    },
    enableSorting: true,
  },
  {
    id: 'classification',
    accessorKey: 'classification',
    header: 'Classification',
    cell: ({ row }: { row: Row<DataAsset> }) => {
      const cls = row.original.classification;
      const color = CLASSIFICATION_COLORS[cls] ?? 'bg-muted text-muted-foreground';
      return (
        <span className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${color}`}>
          {cls.replace(/_/g, ' ')}
        </span>
      );
    },
    enableSorting: true,
  },
  {
    id: 'posture_score',
    accessorKey: 'posture_score',
    header: 'Posture',
    cell: ({ row }: { row: Row<DataAsset> }) => <ScoreBar score={row.original.posture_score} />,
    enableSorting: true,
  },
  {
    id: 'risk_score',
    accessorKey: 'risk_score',
    header: 'Risk',
    cell: ({ row }: { row: Row<DataAsset> }) => <ScoreBar score={row.original.risk_score} invert />,
    enableSorting: true,
  },
  {
    id: 'encrypted',
    header: 'Encrypted',
    cell: ({ row }: { row: Row<DataAsset> }) => {
      const at = row.original.encrypted_at_rest;
      const in_ = row.original.encrypted_in_transit;
      return (
        <div className="flex items-center gap-2 text-xs">
          <span title="At rest">{at ? <Lock className="h-3.5 w-3.5 text-green-500" /> : <XCircle className="h-3.5 w-3.5 text-red-500" />}</span>
          <span title="In transit">{in_ ? <CheckCircle className="h-3.5 w-3.5 text-green-500" /> : <XCircle className="h-3.5 w-3.5 text-red-500" />}</span>
        </div>
      );
    },
  },
  {
    id: 'network_exposure',
    header: 'Exposure',
    cell: ({ row }: { row: Row<DataAsset> }) => {
      const exp = row.original.network_exposure;
      if (!exp) return <span className="text-xs text-muted-foreground">—</span>;
      const isInternet = exp === 'internet';
      return (
        <div className={`flex items-center gap-1 text-xs ${isInternet ? 'text-red-600' : 'text-muted-foreground'}`}>
          {isInternet && <Globe className="h-3.5 w-3.5" />}
          <span className="capitalize">{exp.replace(/_/g, ' ')}</span>
        </div>
      );
    },
  },
  {
    id: 'pii',
    header: 'PII Types',
    cell: ({ row }: { row: Row<DataAsset> }) => {
      const types = row.original.pii_types;
      if (!types.length) return <span className="text-xs text-muted-foreground">None</span>;
      return (
        <div className="flex flex-wrap gap-1">
          {types.slice(0, 2).map((t) => (
            <Badge key={t} variant="outline" className="text-xs px-1.5 py-0">{t}</Badge>
          ))}
          {types.length > 2 && <Badge variant="outline" className="text-xs px-1.5 py-0">+{types.length - 2}</Badge>}
        </div>
      );
    },
  },
  {
    id: 'findings',
    header: 'Findings',
    cell: ({ row }: { row: Row<DataAsset> }) => {
      const count = row.original.posture_findings.length;
      if (!count) return <span className="text-xs text-green-600">✓ Clean</span>;
      return <span className="text-xs font-medium text-orange-600">{count} issue{count !== 1 ? 's' : ''}</span>;
    },
  },
];
