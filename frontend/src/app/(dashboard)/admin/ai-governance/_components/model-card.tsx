'use client';

import { Card, CardContent } from '@/components/ui/card';
import { cn } from '@/lib/utils';

interface ModelCardProps {
  label: string;
  value: string | number;
  helper?: string;
  className?: string;
}

export function ModelCard({ label, value, helper, className }: ModelCardProps) {
  return (
    <Card className={cn('overflow-hidden border-border/70 bg-card/80', className)}>
      <CardContent className="space-y-2 p-5">
        <p className="text-xs font-semibold uppercase tracking-[0.24em] text-muted-foreground">
          {label}
        </p>
        <p className="text-3xl font-semibold tracking-tight">{value}</p>
        {helper ? <p className="text-sm text-muted-foreground">{helper}</p> : null}
      </CardContent>
    </Card>
  );
}
