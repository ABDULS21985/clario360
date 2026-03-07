'use client';

import { useState } from 'react';
import { Plus, Trash2, GripVertical } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import type { CorrelationRuleContent, CorrelationEventType, RuleCondition } from '@/types/cyber';

const SIGMA_OPERATORS = [
  { value: 'exact', label: 'exact' }, { value: '|in', label: '|in' },
  { value: '|contains', label: '|contains' }, { value: '|not contains', label: '|not contains' },
  { value: '|startswith', label: '|startswith' }, { value: '|endswith', label: '|endswith' },
  { value: '|re', label: '|re' }, { value: '|gt', label: '|gt' },
  { value: '|lt', label: '|lt' }, { value: '|gte', label: '|gte' },
  { value: '|lte', label: '|lte' }, { value: '|cidr', label: '|cidr' },
];
const EVENT_FIELDS = [
  'event_type', 'process_name', 'command_line', 'user', 'source_ip',
  'destination_ip', 'destination_port', 'file_path', 'hostname', 'action',
];
const GROUP_BY_FIELDS = ['source_ip', 'user', 'hostname', 'process_name'];

function emptyCondition(): RuleCondition {
  return { field: 'event_type', operator: 'exact', value: '' };
}

function emptyEventType(index: number): CorrelationEventType {
  return { name: `event_${index + 1}`, conditions: [emptyCondition()] };
}

interface RuleCorrelationEditorProps {
  value: CorrelationRuleContent;
  onChange: (value: CorrelationRuleContent) => void;
}

export function RuleCorrelationEditor({ value, onChange }: RuleCorrelationEditorProps) {
  function updateEventType(idx: number, et: CorrelationEventType) {
    const event_types = [...value.event_types];
    event_types[idx] = et;
    // Update sequence if name changed
    const oldName = value.event_types[idx].name;
    const sequence = value.sequence.map((s) => (s === oldName ? et.name : s));
    onChange({ ...value, event_types, sequence });
  }

  function addEventType() {
    const newEt = emptyEventType(value.event_types.length);
    onChange({
      ...value,
      event_types: [...value.event_types, newEt],
      sequence: [...value.sequence, newEt.name],
    });
  }

  function removeEventType(idx: number) {
    const removed = value.event_types[idx].name;
    const event_types = value.event_types.filter((_, i) => i !== idx);
    const sequence = value.sequence.filter((s) => s !== removed);
    onChange({ ...value, event_types, sequence });
  }

  function moveSequenceItem(from: number, to: number) {
    const seq = [...value.sequence];
    const [item] = seq.splice(from, 1);
    seq.splice(to, 0, item);
    onChange({ ...value, sequence: seq });
  }

  function updateCondition(etIdx: number, condIdx: number, cond: RuleCondition) {
    const et = { ...value.event_types[etIdx] };
    const conditions = [...et.conditions];
    conditions[condIdx] = cond;
    updateEventType(etIdx, { ...et, conditions });
  }

  function addCondition(etIdx: number) {
    const et = value.event_types[etIdx];
    updateEventType(etIdx, { ...et, conditions: [...et.conditions, emptyCondition()] });
  }

  function removeCondition(etIdx: number, condIdx: number) {
    const et = value.event_types[etIdx];
    updateEventType(etIdx, { ...et, conditions: et.conditions.filter((_, i) => i !== condIdx) });
  }

  return (
    <div className="space-y-4">
      {/* Event Types */}
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Label className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            Event Types
          </Label>
          <Button type="button" variant="ghost" size="sm" className="h-6 text-xs" onClick={addEventType}>
            <Plus className="mr-1 h-3 w-3" /> Add Event Type
          </Button>
        </div>
        {value.event_types.map((et, etIdx) => (
          <div key={etIdx} className="rounded-lg border p-3 space-y-2">
            <div className="flex items-center gap-2">
              <Input
                value={et.name}
                onChange={(e) => updateEventType(etIdx, { ...et, name: e.target.value })}
                className="h-6 w-28 text-xs font-mono"
                placeholder="event_name"
              />
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="ml-auto h-6 w-6 p-0 text-muted-foreground hover:text-destructive"
                onClick={() => removeEventType(etIdx)}
                disabled={value.event_types.length <= 1}
              >
                <Trash2 className="h-3 w-3" />
              </Button>
            </div>
            {et.conditions.map((cond, condIdx) => (
              <div key={condIdx} className="flex items-center gap-2">
                <Select
                  value={cond.field}
                  onValueChange={(v) => updateCondition(etIdx, condIdx, { ...cond, field: v })}
                >
                  <SelectTrigger className="h-7 w-32 text-xs"><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {EVENT_FIELDS.map((f) => (
                      <SelectItem key={f} value={f} className="text-xs font-mono">{f}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Select
                  value={cond.operator}
                  onValueChange={(v) => updateCondition(etIdx, condIdx, { ...cond, operator: v })}
                >
                  <SelectTrigger className="h-7 w-32 text-xs"><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {SIGMA_OPERATORS.map((op) => (
                      <SelectItem key={op.value} value={op.value} className="text-xs">{op.label}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Input
                  value={cond.value}
                  onChange={(e) => updateCondition(etIdx, condIdx, { ...cond, value: e.target.value })}
                  placeholder="value"
                  className="h-7 flex-1 text-xs font-mono"
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="h-7 w-7 p-0"
                  onClick={() => removeCondition(etIdx, condIdx)}
                  disabled={et.conditions.length <= 1}
                >
                  <Trash2 className="h-3 w-3" />
                </Button>
              </div>
            ))}
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="h-6 text-xs text-primary"
              onClick={() => addCondition(etIdx)}
            >
              <Plus className="mr-1 h-3 w-3" /> Add Condition
            </Button>
          </div>
        ))}
      </div>

      {/* Sequence */}
      <div>
        <Label className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          Sequence (ordered)
        </Label>
        <div className="mt-1.5 space-y-1">
          {value.sequence.map((name, i) => (
            <div key={i} className="flex items-center gap-2 rounded border bg-muted/30 px-2 py-1">
              <GripVertical className="h-3.5 w-3.5 text-muted-foreground" />
              <span className="flex-1 text-xs font-mono">{name}</span>
              <div className="flex gap-0.5">
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="h-5 w-5 p-0 text-muted-foreground"
                  onClick={() => i > 0 && moveSequenceItem(i, i - 1)}
                  disabled={i === 0}
                >
                  ↑
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="h-5 w-5 p-0 text-muted-foreground"
                  onClick={() => i < value.sequence.length - 1 && moveSequenceItem(i, i + 1)}
                  disabled={i === value.sequence.length - 1}
                >
                  ↓
                </Button>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Group By and Window */}
      <div className="grid grid-cols-2 gap-3">
        <div>
          <Label className="text-xs text-muted-foreground">Group By</Label>
          <Select
            value={value.group_by ?? ''}
            onValueChange={(v) => onChange({ ...value, group_by: v || undefined })}
          >
            <SelectTrigger className="mt-1 h-8 text-xs"><SelectValue placeholder="(none)" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="" className="text-xs">(none)</SelectItem>
              {GROUP_BY_FIELDS.map((f) => (
                <SelectItem key={f} value={f} className="text-xs font-mono">{f}</SelectItem>
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
      </div>
    </div>
  );
}

export function defaultCorrelationContent(): CorrelationRuleContent {
  return {
    event_types: [emptyEventType(0), emptyEventType(1)],
    sequence: ['event_1', 'event_2'],
    window: '10m',
  };
}
