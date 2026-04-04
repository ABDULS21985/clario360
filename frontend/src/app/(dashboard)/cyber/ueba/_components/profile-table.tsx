'use client';

import Link from 'next/link';
import { formatDistanceToNow } from 'date-fns';
import { Badge } from '@/components/ui/badge';
import type { UebaRiskRankingItem } from './types';

export function ProfileTable({ items }: { items: UebaRiskRankingItem[] }) {
  return (
    <div className="overflow-hidden rounded-lg border">
      <table className="w-full text-sm">
        <thead className="bg-muted/40 text-left">
          <tr>
            <th className="px-4 py-3 font-medium">Entity</th>
            <th className="px-4 py-3 font-medium">Risk</th>
            <th className="px-4 py-3 font-medium">Maturity</th>
            <th className="px-4 py-3 font-medium">Alerts</th>
            <th className="px-4 py-3 font-medium">Last Seen</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <tr key={item.entity_id} className="border-t">
              <td className="px-4 py-3">
                <Link href={`/cyber/ueba/profiles/${encodeURIComponent(item.entity_id)}`} className="font-medium hover:underline">
                  {item.entity_name}
                </Link>
                <div className="text-xs text-muted-foreground">{item.entity_type.replaceAll('_', ' ')}</div>
              </td>
              <td className="px-4 py-3">
                <Badge variant={item.risk_score >= 75 ? 'destructive' : item.risk_score >= 50 ? 'warning' : 'outline'}>
                  {item.risk_score.toFixed(0)} · {item.risk_level}
                </Badge>
              </td>
              <td className="px-4 py-3">
                <Badge variant={item.profile_maturity === 'mature' ? 'success' : item.profile_maturity === 'baseline' ? 'default' : 'secondary'}>
                  {item.profile_maturity}
                </Badge>
              </td>
              <td className="px-4 py-3">{item.alert_count_7d} / {item.alert_count_30d}</td>
              <td className="px-4 py-3 text-muted-foreground">
                {formatDistanceToNow(new Date(item.last_seen_at), { addSuffix: true })}
              </td>
            </tr>
          ))}
          {items.length === 0 && (
            <tr>
              <td className="px-4 py-6 text-center text-muted-foreground" colSpan={5}>
                No UEBA profiles available.
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
