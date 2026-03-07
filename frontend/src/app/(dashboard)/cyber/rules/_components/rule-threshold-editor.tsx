'use client';

import { Plus, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import type { ThresholdRuleContent, RuleCondition } from '@/types/cyber';

const SIGMA_OPERATORS = [
  { value: 'exact', label: 'exact (=)' },
  { value: '|in', label: '|in' },
  { value: '|contains', label: '|contains' },
  { value: '|not contains', label: '|not contains' },
  { value: '|startswith', label: '|startswith' },
  { value: '|endswith', label: '|endswith' },
  { value: '|re', label: '|re (regex)' },
  { value: '|gt', label: '|gt (>)' },
  { value: '|lt', label: '|lt (<)' },
  { value: '|gte', label: '|gte (>=)' },
  { value: '|lte', label: '|lte (<=)' },
  { value: '|cidr', label: '|cidr' },
];

const EVENT_FIELDS = [
  'event_type', 'process_name', 'command_line', 'user', 'source_ip',
  'destination_ip', 'destination_port', 'file_path', 'hostname', 'action',
  'status', 'bytes', 'protocol', 'error_code',
];

function emptyCondition(): RuleCondition {
  return { field: 'event_type', operator: 'exact', value: '' };
}

interface RuleThresholdEditorProps {
  value: ThresholdRuleContent;
  onChange: (value: ThresholdRuleContent) => void;
}

export function RuleThresholdEditor({ value, onChange }: RuleThresholdEditorProps) {
  function updateCondition(idx: number, cond: RuleCondition) {
    const conditions = [...value.filter_conditions];
    conditions[idx] = cond;
    onChange({ ...value, filter_conditions: conditions });
  }

  function addCondition() {
    onChange({ ...value, filter_conditions: [...value.filter_conditions, emptyCondition()] });
  }

  function removeCondition(idx: number) {
    onChange({ ...value, filter_conditions: value.filter_conditions.filter((_, i) => i !== idx) });
  }

  return (
    <div className="space-y-4">
      {/* Filter conditions */}
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Label className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            Filter Conditions
          </Label>
          <Button type="button" variant="ghost" size="sm" className="h-6 text-xs" onClick={addCondition}>
            <Plus className="mr-1 h-3 w-3" /> Add Condition
          </Button>
        </div>
        {value.filter_conditions.map((cond, idx) => (
          <div key={idx} className="flex items-center gap-2">
            <Select
              value={cond.field}
              onValueChange={(v) => updateCondition(idx, { ...cond, field: v })}
            >
              <SelectTrigger className="h-7 w-36 text-xs"><SelectValue /></SelectTrigger>
              <SelectContent>
                {EVENT_FIELDS.map((f) => (
                  <SelectItem key={f} value={f} className="text-xs font-mono">{f}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select
              value={cond.operator}
              onValueChange={(v) => updateCondition(idx, { ...cond, operator: v })}
            >
              <SelectTrigger className="h-7 w-36 text-xs"><SelectValue /></SelectTrigger>
              <SelectContent>
                {SIGMA_OPERATORS.map((op) => (
                  <SelectItem key={op.value} value={op.value} className="text-xs">{op.label}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Input
              value={cond.value}
              onChange={(e) => updateCondition(idx, { ...cond, value: e.target.value })}
              placeholder="value"
              className="h-7 flex-1 text-xs font-mono"
            />
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="h-7 w-7 p-0 text-muted-foreground hover:text-destructive"
              onClick={() => removeCondition(idx)}
              disabled={value.filter_conditions.length <= 1}
            >
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          </div>
        ))}
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div>
          <Label className="text-xs text-muted-foreground">Group By Field</Label>
          <Select
            value={value.group_by ?? ''}
            onValueChange={(v) => onChange({ ...value, group_by: v || undefined })}
          >
            <SelectTrigger className="mt-1 h-8 text-xs"><SelectValue placeholder="(none)" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="" className="text-xs">(none)</SelectItem>
              {EVENT_FIELDS.map((f) => (
                <SelectItem key={f} value={f} className="text-xs font-mono">{f}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div>
          <Label className="text-xs text-muted-foreground">Metric</Label>
          <Select
            value={value.metric}
            onValueChange={(v) => onChange({ ...value, metric: v as ThresholdRuleContent['metric'] })}
          >
            <SelectTrigger className="mt-1 h-8 text-xs"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="count" className="text-xs">count</SelectItem>
              <SelectItem value="sum" className="text-xs">sum</SelectItem>
              <SelectItem value="distinct" className="text-xs">distinct</SelectItem>
            </SelectContent>
          </Select>
        </div>
        {(value.metric === 'sum' || value.metric === 'distinct') && (
          <div>
            <Label className="text-xs text-muted-foreground">Metric Field</Label>
            <Select
              value={value.metric_field ?? ''}
              onValueChange={(v) => onChange({ ...value, metric_field: v || undefined })}
            >
              <SelectTrigger className="mt-1 h-8 text-xs"><SelectValue placeholder="Select field" /></SelectTrigger>
              <SelectContent>
                {EVENT_FIELDS.map((f) => (
                  <SelectItem key={f} value={f} className="text-xs font-mono">{f}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}
        <div>
          <Label className="text-xs text-muted-foreground">Threshold Value</Label>
          <Input
            type="number"
            min={1}
            value={value.threshold}
            onChange={(e) => onChange({ ...value, threshold: parseInt(e.target.value) || 0 })}
            className="mt-1 h-8 text-xs"
            placeholder="e.g. 5"
          />
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
      </div>
    </div>
  );
}

export function defaultThresholdContent(): ThresholdRuleContent {
  return {
    filter_conditions: [{ field: 'event_type', operator: 'exact', value: '' }],
    metric: 'count',
    threshold: 5,
    window: '5m',
  };
}
