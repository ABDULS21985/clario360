'use client';

import { Badge } from '@/components/ui/badge';
import type { UebaAlert } from './types';

export function SignalEvidenceViewer({ alert }: { alert: UebaAlert }) {
  const signals = alert.triggering_signals ?? [];
  return (
    <div className="space-y-3">
      {signals.map((signal) => (
        <div key={`${alert.id}-${signal.event_id}-${signal.signal_type}`} className="rounded-lg border p-3">
          <div className="mb-2 flex items-center justify-between gap-2">
            <div>
              <div className="font-medium">{signal.title}</div>
              <div className="text-xs text-muted-foreground">{signal.signal_type.replaceAll('_', ' ')}</div>
            </div>
            <Badge variant={signal.severity === 'critical' ? 'destructive' : signal.severity === 'high' ? 'warning' : 'outline'}>
              {signal.severity}
            </Badge>
          </div>
          <div className="grid gap-2 text-xs text-muted-foreground">
            <div>{signal.description}</div>
            <div>Expected: {signal.expected_value}</div>
            <div>Actual: {signal.actual_value}</div>
            <div>Confidence: {(signal.confidence * 100).toFixed(0)}%</div>
            <div>Event ID: <span className="font-mono">{signal.event_id}</span></div>
          </div>
        </div>
      ))}
      {signals.length === 0 && <p className="text-sm text-muted-foreground">No structured evidence attached.</p>}
    </div>
  );
}
