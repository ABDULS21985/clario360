'use client';

import { useRouter } from 'next/navigation';
import { Badge } from '@/components/ui/badge';
import type { IdentityProfile } from '@/types/cyber';

interface IdentityRiskTableProps {
  identities: IdentityProfile[];
}

function riskScoreColor(score: number): string {
  if (score >= 75) return 'text-red-600 font-semibold';
  if (score >= 50) return 'text-orange-600 font-semibold';
  if (score >= 25) return 'text-amber-600';
  return 'text-green-600';
}

function blastRadiusColor(score: number): string {
  if (score >= 75) return 'text-red-600 font-semibold';
  if (score >= 50) return 'text-orange-600 font-semibold';
  if (score >= 25) return 'text-amber-600';
  return 'text-green-600';
}

function formatIdentityType(type: string): string {
  return type
    .split('_')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}

function statusBadgeVariant(status: string) {
  switch (status) {
    case 'active':
      return 'success' as const;
    case 'inactive':
      return 'secondary' as const;
    case 'under_review':
      return 'warning' as const;
    case 'remediated':
      return 'default' as const;
    default:
      return 'outline' as const;
  }
}

export function IdentityRiskTable({ identities }: IdentityRiskTableProps) {
  const router = useRouter();

  if (identities.length === 0) {
    return (
      <div className="rounded-lg border bg-muted/20 p-6 text-center text-sm text-muted-foreground">
        No identity profiles available.
      </div>
    );
  }

  return (
    <div className="overflow-x-auto rounded-xl border bg-card">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b text-left">
            <th className="px-4 py-3 font-medium text-muted-foreground">Name</th>
            <th className="px-4 py-3 font-medium text-muted-foreground">Type</th>
            <th className="px-4 py-3 font-medium text-muted-foreground">Risk Score</th>
            <th className="px-4 py-3 font-medium text-muted-foreground">Blast Radius</th>
            <th className="px-4 py-3 font-medium text-muted-foreground">Overprivileged</th>
            <th className="px-4 py-3 font-medium text-muted-foreground">Status</th>
          </tr>
        </thead>
        <tbody>
          {identities.map((identity) => (
            <tr
              key={identity.id}
              className="cursor-pointer border-b last:border-0 transition-colors hover:bg-muted/50"
              onClick={() => router.push(`/cyber/dspm/access/identities/${identity.id}`)}
            >
              <td className="px-4 py-3">
                <div>
                  <p className="font-medium">{identity.identity_name}</p>
                  <p className="text-xs text-muted-foreground">{identity.identity_email}</p>
                </div>
              </td>
              <td className="px-4 py-3">
                <Badge variant="outline">{formatIdentityType(identity.identity_type)}</Badge>
              </td>
              <td className="px-4 py-3">
                <span className={`tabular-nums ${riskScoreColor(identity.access_risk_score)}`}>
                  {Math.round(identity.access_risk_score)}
                </span>
              </td>
              <td className="px-4 py-3">
                <span className={`tabular-nums ${blastRadiusColor(identity.blast_radius_score)}`}>
                  {Math.round(identity.blast_radius_score)}
                </span>
              </td>
              <td className="px-4 py-3">
                {identity.overprivileged_count > 0 ? (
                  <Badge variant="destructive">{identity.overprivileged_count}</Badge>
                ) : (
                  <span className="text-xs text-muted-foreground">None</span>
                )}
              </td>
              <td className="px-4 py-3">
                <Badge variant={statusBadgeVariant(identity.status)}>
                  {formatIdentityType(identity.status)}
                </Badge>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
