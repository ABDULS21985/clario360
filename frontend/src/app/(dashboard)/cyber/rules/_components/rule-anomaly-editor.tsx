'use client';

import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { RULE_FIELD_OPTIONS } from '@/lib/cyber-rules';
import type { AnomalyRuleContent } from '@/types/cyber';

const ANOMALY_METRICS = [
  'event_count', 'unique_ips', 'bytes_transferred', 'login_failures',
  'process_count', 'dns_queries', 'connection_count', 'error_rate',
];

interface RuleAnomalyEditorProps {
  value: AnomalyRuleContent;
  onChange: (value: AnomalyRuleContent) => void;
}

export function RuleAnomalyEditor({ value, onChange }: RuleAnomalyEditorProps) {
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div>
          <Label className="text-xs text-muted-foreground">Metric</Label>
          <Select
            value={value.metric}
            onValueChange={(v) => onChange({ ...value, metric: v })}
          >
            <SelectTrigger className="mt-1 h-8 text-xs"><SelectValue /></SelectTrigger>
            <SelectContent>
              {ANOMALY_METRICS.map((m) => (
                <SelectItem key={m} value={m} className="text-xs">{m.replace(/_/g, ' ')}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div>
          <Label className="text-xs text-muted-foreground">Group By Field</Label>
          <Select
            value={value.group_by ?? ''}
            onValueChange={(v) => onChange({ ...value, group_by: v || undefined })}
          >
            <SelectTrigger className="mt-1 h-8 text-xs"><SelectValue placeholder="(none)" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="" className="text-xs">(none)</SelectItem>
              {RULE_FIELD_OPTIONS.map((field) => (
                <SelectItem key={field.value} value={field.value} className="text-xs">
                  {field.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div>
          <Label className="text-xs text-muted-foreground">Time Window</Label>
          <Input
            value={value.window}
            onChange={(e) => onChange({ ...value, window: e.target.value })}
            className="mt-1 h-8 text-xs"
            placeholder="5m, 1h, 24h"
          />
        </div>
        <div>
          <Label className="text-xs text-muted-foreground">Z-Score Threshold</Label>
          <Input
            type="number"
            step={0.1}
            min={0.5}
            value={value.z_score_threshold}
            onChange={(e) => onChange({ ...value, z_score_threshold: parseFloat(e.target.value) || 3 })}
            className="mt-1 h-8 text-xs"
          />
        </div>
        <div>
          <Label className="text-xs text-muted-foreground">Min Baseline Samples</Label>
          <Input
            type="number"
            min={1}
            value={value.min_baseline_samples}
            onChange={(e) =>
              onChange({ ...value, min_baseline_samples: parseInt(e.target.value) || 100 })
            }
            className="mt-1 h-8 text-xs"
          />
        </div>
        <div>
          <Label className="text-xs text-muted-foreground">Direction</Label>
          <Select
            value={value.direction}
            onValueChange={(v) => onChange({ ...value, direction: v as AnomalyRuleContent['direction'] })}
          >
            <SelectTrigger className="mt-1 h-8 text-xs"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="above" className="text-xs">above (spike)</SelectItem>
              <SelectItem value="below" className="text-xs">below (drop)</SelectItem>
              <SelectItem value="both" className="text-xs">both (any deviation)</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>
    </div>
  );
}

export function defaultAnomalyContent(): AnomalyRuleContent {
  return {
    metric: 'event_count',
    window: '1h',
    z_score_threshold: 3.0,
    min_baseline_samples: 100,
    direction: 'above',
  };
}
