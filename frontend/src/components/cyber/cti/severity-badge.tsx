'use client';

import { Badge } from '@/components/ui/badge';
import { CTI_SEVERITY_COLORS } from '@/types/cti';

interface SeverityBadgeProps {
  severity: string;
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

const SIZE_STYLES: Record<NonNullable<SeverityBadgeProps['size']>, string> = {
  sm: 'px-1.5 py-0 text-[10px]',
  md: 'px-2 py-0.5 text-xs',
  lg: 'px-2.5 py-1 text-sm',
};

export function CTISeverityBadge({ severity, size = 'md', className }: SeverityBadgeProps) {
  const color = CTI_SEVERITY_COLORS[severity] ?? '#6B7280';
  return (
    <Badge
      className={`${SIZE_STYLES[size]} ${className ?? ''}`.trim()}
      style={{ backgroundColor: color, color: severity === 'medium' ? '#000' : '#fff', border: 'none' }}
    >
      {severity.charAt(0).toUpperCase() + severity.slice(1)}
    </Badge>
  );
}
