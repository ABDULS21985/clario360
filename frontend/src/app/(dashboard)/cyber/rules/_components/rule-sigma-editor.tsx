'use client';

import { useState, useEffect, useCallback } from 'react';
import { Plus, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Label } from '@/components/ui/label';
import type { SigmaRuleContent, RuleSelection, RuleCondition } from '@/types/cyber';

const SIGMA_OPERATORS = [
  { value: 'exact', label: 'exact (=)' },
  { value: '|in', label: '|in (in list)' },
  { value: '|contains', label: '|contains' },
  { value: '|not contains', label: '|not contains' },
  { value: '|startswith', label: '|startswith' },
  { value: '|endswith', label: '|endswith' },
  { value: '|re', label: '|re (regex)' },
  { value: '|gt', label: '|gt (>)' },
  { value: '|lt', label: '|lt (<)' },
  { value: '|gte', label: '|gte (>=)' },
  { value: '|lte', label: '|lte (<=)' },
  { value: '|cidr', label: '|cidr (CIDR range)' },
];

const EVENT_FIELDS = [
  'event_type', 'process_name', 'command_line', 'user', 'source_ip',
  'destination_ip', 'destination_port', 'file_path', 'registry_key',
  'hostname', 'action', 'status', 'bytes', 'protocol',
];

function emptyCondition(): RuleCondition {
  return { field: 'event_type', operator: 'exact', value: '' };
}

function emptySelection(index: number): RuleSelection {
  return { name: `selection_${index + 1}`, conditions: [emptyCondition()] };
}

interface ConditionRowProps {
  cond: RuleCondition;
  onChange: (cond: RuleCondition) => void;
  onRemove: () => void;
  canRemove: boolean;
}

function ConditionRow({ cond, onChange, onRemove, canRemove }: ConditionRowProps) {
  return (
    <div className="flex items-center gap-2">
      <Select
        value={cond.field}
        onValueChange={(v) => onChange({ ...cond, field: v })}
      >
        <SelectTrigger className="h-7 w-36 text-xs">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {EVENT_FIELDS.map((f) => (
            <SelectItem key={f} value={f} className="text-xs font-mono">{f}</SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Select
        value={cond.operator}
        onValueChange={(v) => onChange({ ...cond, operator: v })}
      >
        <SelectTrigger className="h-7 w-40 text-xs">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {SIGMA_OPERATORS.map((op) => (
            <SelectItem key={op.value} value={op.value} className="text-xs">{op.label}</SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Input
        value={cond.value}
        onChange={(e) => onChange({ ...cond, value: e.target.value })}
        placeholder="value"
        className="h-7 flex-1 text-xs font-mono"
      />
      <Button
        type="button"
        variant="ghost"
        size="sm"
        className="h-7 w-7 p-0 text-muted-foreground hover:text-destructive"
        onClick={onRemove}
        disabled={!canRemove}
        aria-label="Remove condition"
      >
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </div>
  );
}

interface SelectionBlockProps {
  selection: RuleSelection;
  onChange: (sel: RuleSelection) => void;
  onRemove: () => void;
  canRemove: boolean;
  label: string;
}

function SelectionBlock({ selection, onChange, onRemove, canRemove, label }: SelectionBlockProps) {
  function updateCondition(idx: number, cond: RuleCondition) {
    const conditions = [...selection.conditions];
    conditions[idx] = cond;
    onChange({ ...selection, conditions });
  }

  function addCondition() {
    onChange({ ...selection, conditions: [...selection.conditions, emptyCondition()] });
  }

  function removeCondition(idx: number) {
    onChange({ ...selection, conditions: selection.conditions.filter((_, i) => i !== idx) });
  }

  return (
    <div className="rounded-lg border p-3 space-y-2">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-xs font-medium text-muted-foreground uppercase">{label}</span>
          <Input
            value={selection.name}
            onChange={(e) => onChange({ ...selection, name: e.target.value })}
            className="h-6 w-32 text-xs font-mono"
            placeholder="selection_name"
          />
        </div>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-6 w-6 p-0 text-muted-foreground hover:text-destructive"
          onClick={onRemove}
          disabled={!canRemove}
          aria-label="Remove selection"
        >
          <Trash2 className="h-3 w-3" />
        </Button>
      </div>
      <div className="space-y-1.5">
        {selection.conditions.map((cond, idx) => (
          <ConditionRow
            key={idx}
            cond={cond}
            onChange={(c) => updateCondition(idx, c)}
            onRemove={() => removeCondition(idx)}
            canRemove={selection.conditions.length > 1}
          />
        ))}
      </div>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        className="h-6 text-xs text-primary"
        onClick={addCondition}
      >
        <Plus className="mr-1 h-3 w-3" /> Add Condition
      </Button>
    </div>
  );
}

interface RuleSigmaEditorProps {
  value: SigmaRuleContent;
  onChange: (value: SigmaRuleContent) => void;
}

export function RuleSigmaEditor({ value, onChange }: RuleSigmaEditorProps) {
  function updateSelection(idx: number, sel: RuleSelection) {
    const selections = [...value.selections];
    selections[idx] = sel;
    onChange({ ...value, selections });
  }

  function addSelection() {
    onChange({
      ...value,
      selections: [...value.selections, emptySelection(value.selections.length)],
    });
  }

  function removeSelection(idx: number) {
    onChange({ ...value, selections: value.selections.filter((_, i) => i !== idx) });
  }

  function updateFilter(idx: number, sel: RuleSelection) {
    const filters = [...(value.filters ?? [])];
    filters[idx] = sel;
    onChange({ ...value, filters });
  }

  function addFilter() {
    onChange({
      ...value,
      filters: [...(value.filters ?? []), emptySelection((value.filters?.length ?? 0))],
    });
  }

  function removeFilter(idx: number) {
    onChange({ ...value, filters: (value.filters ?? []).filter((_, i) => i !== idx) });
  }

  return (
    <div className="space-y-4">
      {/* Selections */}
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Label className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            Selections
          </Label>
          <Button type="button" variant="ghost" size="sm" className="h-6 text-xs" onClick={addSelection}>
            <Plus className="mr-1 h-3 w-3" /> Add Selection
          </Button>
        </div>
        {value.selections.map((sel, idx) => (
          <SelectionBlock
            key={idx}
            selection={sel}
            onChange={(s) => updateSelection(idx, s)}
            onRemove={() => removeSelection(idx)}
            canRemove={value.selections.length > 1}
            label="Selection"
          />
        ))}
      </div>

      {/* Filters */}
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Label className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            Filters (exclusions)
          </Label>
          <Button type="button" variant="ghost" size="sm" className="h-6 text-xs" onClick={addFilter}>
            <Plus className="mr-1 h-3 w-3" /> Add Filter
          </Button>
        </div>
        {(value.filters ?? []).map((f, idx) => (
          <SelectionBlock
            key={idx}
            selection={f}
            onChange={(s) => updateFilter(idx, s)}
            onRemove={() => removeFilter(idx)}
            canRemove={true}
            label="Filter"
          />
        ))}
      </div>

      {/* Condition */}
      <div>
        <Label className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          Condition Expression
          <Tooltip text="Boolean expression: e.g. (selection_main or selection_alt) and not filter_exclude" />
        </Label>
        <Input
          value={value.condition}
          onChange={(e) => onChange({ ...value, condition: e.target.value })}
          placeholder="e.g. selection_main and not filter_exclude"
          className="mt-1 font-mono text-xs"
        />
      </div>

      {/* Timeframe and Threshold */}
      <div className="grid grid-cols-2 gap-3">
        <div>
          <Label className="text-xs text-muted-foreground">Timeframe (optional)</Label>
          <Input
            value={value.timeframe ?? ''}
            onChange={(e) => onChange({ ...value, timeframe: e.target.value || undefined })}
            placeholder="5m, 1h, 24h"
            className="mt-1 text-xs"
          />
        </div>
        <div>
          <Label className="text-xs text-muted-foreground">Count Threshold (optional)</Label>
          <Input
            type="number"
            min={1}
            value={value.threshold ?? ''}
            onChange={(e) =>
              onChange({ ...value, threshold: e.target.value ? parseInt(e.target.value) : undefined })
            }
            placeholder="e.g. 3"
            className="mt-1 text-xs"
          />
        </div>
      </div>
    </div>
  );
}

// Tiny inline tooltip
function Tooltip({ text }: { text: string }) {
  return (
    <span className="ml-1 cursor-help text-muted-foreground" title={text}>
      ⓘ
    </span>
  );
}

export function defaultSigmaContent(): SigmaRuleContent {
  return {
    selections: [emptySelection(0)],
    filters: [],
    condition: 'selection_1',
  };
}
