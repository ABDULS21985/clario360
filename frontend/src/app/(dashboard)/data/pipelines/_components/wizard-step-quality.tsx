'use client';

import { Plus } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import type { PipelineQualityValues } from '@/app/(dashboard)/data/pipelines/_components/pipeline-wizard-types';
import { pipelineQualityGateSchema } from '@/app/(dashboard)/data/pipelines/_components/pipeline-wizard-types';
import { createEmptyQualityGate } from '@/app/(dashboard)/data/pipelines/_components/pipeline-wizard-utils';

interface WizardStepQualityProps {
  value: PipelineQualityValues;
  availableColumns: string[];
  onBack: () => void;
  onChange: (value: PipelineQualityValues) => void;
  onContinue: () => void;
}

export function WizardStepQuality({
  value,
  availableColumns,
  onBack,
  onChange,
  onContinue,
}: WizardStepQualityProps) {
  const validationErrors = value.quality_gates.map((gate) => pipelineQualityGateSchema.safeParse(gate));
  const hasErrors = validationErrors.some((result) => !result.success);

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3 rounded-xl border bg-muted/10 p-4">
        <Switch
          checked={value.fail_on_quality_gate}
          onCheckedChange={(checked) => onChange({ ...value, fail_on_quality_gate: checked })}
        />
        <div>
          <div className="font-medium">Fail pipeline on quality gate failure</div>
          <div className="text-sm text-muted-foreground">
            Stop the load phase if a gate returns a failed status.
          </div>
        </div>
      </div>

      <div className="space-y-4">
        {value.quality_gates.map((gate, index) => {
          const parsed = validationErrors[index];
          const messages = parsed.success ? [] : parsed.error.issues.map((issue) => issue.message);

          return (
            <div key={gate.id} className="rounded-xl border bg-card p-4">
              <div className="mb-4 flex items-center justify-between">
                <div className="font-medium">Gate {index + 1}</div>
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() =>
                    onChange({
                      ...value,
                      quality_gates: value.quality_gates.filter((item) => item.id !== gate.id),
                    })
                  }
                >
                  Remove
                </Button>
              </div>

              <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                <Input
                  value={gate.name}
                  onChange={(event) =>
                    onChange({
                      ...value,
                      quality_gates: value.quality_gates.map((item) =>
                        item.id === gate.id ? { ...item, name: event.target.value } : item,
                      ),
                    })
                  }
                  placeholder="Gate name"
                />

                <Select
                  value={gate.metric}
                  onValueChange={(next) =>
                    onChange({
                      ...value,
                      quality_gates: value.quality_gates.map((item) =>
                        item.id === gate.id
                          ? {
                              ...item,
                              metric: next as typeof gate.metric,
                            }
                          : item,
                      ),
                    })
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="null_percentage">Null percentage</SelectItem>
                    <SelectItem value="unique_percentage">Unique percentage</SelectItem>
                    <SelectItem value="row_count_change">Row count change</SelectItem>
                    <SelectItem value="min_row_count">Minimum row count</SelectItem>
                    <SelectItem value="custom">Custom expression</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="mt-4 grid grid-cols-1 gap-4 md:grid-cols-2">
                <Select
                  value={gate.column || '__none__'}
                  onValueChange={(next) =>
                    onChange({
                      ...value,
                      quality_gates: value.quality_gates.map((item) =>
                        item.id === gate.id ? { ...item, column: next === '__none__' ? '' : next } : item,
                      ),
                    })
                  }
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Column" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="__none__">No column</SelectItem>
                    {availableColumns.map((column) => (
                      <SelectItem key={column} value={column}>
                        {column}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>

                <Select
                  value={gate.severity}
                  onValueChange={(next) =>
                    onChange({
                      ...value,
                      quality_gates: value.quality_gates.map((item) =>
                        item.id === gate.id
                          ? { ...item, severity: next as typeof gate.severity }
                          : item,
                      ),
                    })
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="critical">Critical</SelectItem>
                    <SelectItem value="high">High</SelectItem>
                    <SelectItem value="medium">Medium</SelectItem>
                    <SelectItem value="low">Low</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="mt-4 grid grid-cols-1 gap-4 md:grid-cols-3">
                <Input
                  value={gate.operator || ''}
                  onChange={(event) =>
                    onChange({
                      ...value,
                      quality_gates: value.quality_gates.map((item) =>
                        item.id === gate.id ? { ...item, operator: event.target.value } : item,
                      ),
                    })
                  }
                  placeholder="Operator"
                />
                <Input
                  type="number"
                  value={gate.threshold ?? ''}
                  onChange={(event) =>
                    onChange({
                      ...value,
                      quality_gates: value.quality_gates.map((item) =>
                        item.id === gate.id
                          ? { ...item, threshold: event.target.value === '' ? undefined : Number(event.target.value) }
                          : item,
                      ),
                    })
                  }
                  placeholder="Threshold"
                />
                <Input
                  value={gate.expression || ''}
                  onChange={(event) =>
                    onChange({
                      ...value,
                      quality_gates: value.quality_gates.map((item) =>
                        item.id === gate.id ? { ...item, expression: event.target.value } : item,
                      ),
                    })
                  }
                  placeholder="Expression"
                />
              </div>

              {messages.length > 0 ? (
                <div className="mt-3 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
                  {messages.join(' ')}
                </div>
              ) : null}
            </div>
          );
        })}
      </div>

      <Button
        type="button"
        variant="outline"
        onClick={() =>
          onChange({
            ...value,
            quality_gates: [...value.quality_gates, createEmptyQualityGate()],
          })
        }
      >
        <Plus className="mr-2 h-4 w-4" />
        Add quality gate
      </Button>

      <div className="flex justify-between">
        <Button type="button" variant="outline" onClick={onBack}>
          Back
        </Button>
        <Button type="button" onClick={onContinue} disabled={hasErrors}>
          Continue
        </Button>
      </div>
    </div>
  );
}

