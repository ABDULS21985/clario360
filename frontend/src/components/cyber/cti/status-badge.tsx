'use client';

import { Badge } from '@/components/ui/badge';
import {
  CTI_CAMPAIGN_STATUS_LABELS,
  CTI_STATUS_COLORS,
  CTI_TAKEDOWN_STATUS_LABELS,
} from '@/types/cti';

interface StatusBadgeProps {
  status: string;
  type: 'campaign' | 'takedown';
  className?: string;
}

export function CTIStatusBadge({ status, type, className }: StatusBadgeProps) {
  const labels = type === 'campaign' ? CTI_CAMPAIGN_STATUS_LABELS : CTI_TAKEDOWN_STATUS_LABELS;
  const label = labels[status] ?? status;
  const color = CTI_STATUS_COLORS[status] ?? '#6B7280';

  return (
    <Badge variant="outline" className={className} style={{ borderColor: color, color }}>
      <span className="mr-1.5 inline-block h-2 w-2 rounded-full" style={{ backgroundColor: color }} />
      {label}
    </Badge>
  );
}
