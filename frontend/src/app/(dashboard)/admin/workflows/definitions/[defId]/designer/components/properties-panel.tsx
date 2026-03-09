'use client';

import { useCallback } from 'react';
import { X, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type {
  WorkflowStep,
  WorkflowStepConfig,
  AssigneeStrategy,
} from '@/types/models';

interface PropertiesPanelProps {
  step: WorkflowStep;
  onUpdate: (updates: Partial<WorkflowStep>) => void;
  onRemove: () => void;
  onClose: () => void;
  readOnly: boolean;
}

export function PropertiesPanel({
  step,
  onUpdate,
  onRemove,
  onClose,
  readOnly,
}: PropertiesPanelProps) {
  const updateConfig = useCallback(
    (configUpdates: Partial<WorkflowStepConfig>) => {
      onUpdate({ config: { ...step.config, ...configUpdates } });
    },
    [step.config, onUpdate],
  );

  const updateAssignee = useCallback(
    (strategyType: string) => {
      const strategies: Record<string, AssigneeStrategy> = {
        specific_user: { type: 'specific_user', user_id: '' },
        role: { type: 'role', role_id: '' },
        manager_of: { type: 'manager_of', relative_to: 'initiator' },
        round_robin: { type: 'round_robin', user_pool: [] },
        least_loaded: { type: 'least_loaded', role_id: '' },
      };
      onUpdate({ assignee_strategy: strategies[strategyType] ?? strategies.role });
    },
    [onUpdate],
  );

  const isHuman = ['approval', 'review', 'task'].includes(step.type);

  return (
    <div className="w-72 border-l bg-background overflow-y-auto">
      {/* Header */}
      <div className="flex items-center justify-between p-3 border-b">
        <h3 className="text-sm font-semibold">Properties</h3>
        <Button variant="ghost" size="icon" className="h-7 w-7" onClick={onClose}>
          <X className="h-4 w-4" />
        </Button>
      </div>

      <div className="p-3 space-y-4">
        {/* Name */}
        <div className="space-y-1.5">
          <Label htmlFor="step-name" className="text-xs">Name</Label>
          <Input
            id="step-name"
            value={step.name}
            onChange={(e) => onUpdate({ name: e.target.value })}
            disabled={readOnly}
            className="h-8 text-sm"
          />
        </div>

        {/* Type (read-only) */}
        <div className="space-y-1.5">
          <Label className="text-xs">Type</Label>
          <div className="text-sm text-muted-foreground capitalize">
            {step.type.replace(/_/g, ' ')}
          </div>
        </div>

        {/* ── Type-specific config ── */}

        {/* Approval config */}
        {step.type === 'approval' && (
          <>
            <div className="space-y-1.5">
              <Label htmlFor="approval-type" className="text-xs">Approval Type</Label>
              <Select
                value={step.config.approval_type ?? 'single'}
                onValueChange={(v) =>
                  updateConfig({ approval_type: v as 'single' | 'unanimous' | 'majority' })
                }
                disabled={readOnly}
              >
                <SelectTrigger id="approval-type" className="h-8 text-sm">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="single">Single Approver</SelectItem>
                  <SelectItem value="unanimous">Unanimous</SelectItem>
                  <SelectItem value="majority">Majority</SelectItem>
                </SelectContent>
              </Select>
            </div>
            {(step.config.approval_type === 'majority' ||
              step.config.approval_type === 'unanimous') && (
              <div className="space-y-1.5">
                <Label htmlFor="min-approvers" className="text-xs">Min Approvers</Label>
                <Input
                  id="min-approvers"
                  type="number"
                  min={1}
                  value={step.config.min_approvers ?? 1}
                  onChange={(e) =>
                    updateConfig({ min_approvers: parseInt(e.target.value, 10) || 1 })
                  }
                  disabled={readOnly}
                  className="h-8 text-sm"
                />
              </div>
            )}
          </>
        )}

        {/* Notification config */}
        {step.type === 'notification' && (
          <>
            <div className="space-y-1.5">
              <Label htmlFor="notif-template" className="text-xs">Template</Label>
              <Input
                id="notif-template"
                value={step.config.notification_template ?? ''}
                onChange={(e) =>
                  updateConfig({ notification_template: e.target.value })
                }
                placeholder="Template name or ID"
                disabled={readOnly}
                className="h-8 text-sm"
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs">Channels</Label>
              <div className="flex flex-wrap gap-1">
                {(['email', 'in_app', 'webhook'] as const).map((ch) => {
                  const active = step.config.notification_channels?.includes(ch);
                  return (
                    <button
                      key={ch}
                      type="button"
                      className={`px-2 py-0.5 text-xs rounded-full border ${
                        active
                          ? 'bg-primary text-primary-foreground'
                          : 'bg-muted text-muted-foreground'
                      }`}
                      disabled={readOnly}
                      onClick={() => {
                        const current = step.config.notification_channels ?? [];
                        updateConfig({
                          notification_channels: active
                            ? current.filter((c) => c !== ch)
                            : [...current, ch],
                        });
                      }}
                    >
                      {ch.replace('_', ' ')}
                    </button>
                  );
                })}
              </div>
            </div>
          </>
        )}

        {/* Delay config */}
        {step.type === 'delay' && (
          <div className="space-y-1.5">
            <Label htmlFor="delay-minutes" className="text-xs">Delay (minutes)</Label>
            <Input
              id="delay-minutes"
              type="number"
              min={1}
              value={step.config.delay_minutes ?? 60}
              onChange={(e) =>
                updateConfig({ delay_minutes: parseInt(e.target.value, 10) || 60 })
              }
              disabled={readOnly}
              className="h-8 text-sm"
            />
          </div>
        )}

        {/* Webhook config */}
        {step.type === 'webhook' && (
          <>
            <div className="space-y-1.5">
              <Label htmlFor="webhook-url" className="text-xs">URL</Label>
              <Input
                id="webhook-url"
                value={step.config.webhook_url ?? ''}
                onChange={(e) => updateConfig({ webhook_url: e.target.value })}
                placeholder="https://..."
                disabled={readOnly}
                className="h-8 text-sm"
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="webhook-method" className="text-xs">Method</Label>
              <Select
                value={step.config.webhook_method ?? 'POST'}
                onValueChange={(v) =>
                  updateConfig({ webhook_method: v as 'GET' | 'POST' | 'PUT' })
                }
                disabled={readOnly}
              >
                <SelectTrigger id="webhook-method" className="h-8 text-sm">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="GET">GET</SelectItem>
                  <SelectItem value="POST">POST</SelectItem>
                  <SelectItem value="PUT">PUT</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="webhook-body" className="text-xs">Body Template</Label>
              <Textarea
                id="webhook-body"
                value={step.config.webhook_body_template ?? ''}
                onChange={(e) =>
                  updateConfig({ webhook_body_template: e.target.value })
                }
                disabled={readOnly}
                rows={3}
                className="text-sm font-mono"
              />
            </div>
          </>
        )}

        {/* Sub-workflow config */}
        {step.type === 'sub_workflow' && (
          <div className="space-y-1.5">
            <Label htmlFor="sub-workflow-id" className="text-xs">Sub-workflow ID</Label>
            <Input
              id="sub-workflow-id"
              value={step.config.sub_workflow_id ?? ''}
              onChange={(e) =>
                updateConfig({ sub_workflow_id: e.target.value })
              }
              disabled={readOnly}
              className="h-8 text-sm"
            />
          </div>
        )}

        {/* Script config */}
        {step.type === 'script' && (
          <div className="space-y-1.5">
            <Label htmlFor="script-id" className="text-xs">Script ID</Label>
            <Input
              id="script-id"
              value={step.config.script_id ?? ''}
              onChange={(e) => updateConfig({ script_id: e.target.value })}
              disabled={readOnly}
              className="h-8 text-sm"
            />
          </div>
        )}

        {/* ── Assignee Strategy (human steps only) ── */}
        {isHuman && (
          <div className="space-y-1.5">
            <Label htmlFor="assignee-strategy" className="text-xs">Assignee Strategy</Label>
            <Select
              value={step.assignee_strategy.type}
              onValueChange={updateAssignee}
              disabled={readOnly}
            >
              <SelectTrigger id="assignee-strategy" className="h-8 text-sm">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="specific_user">Specific User</SelectItem>
                <SelectItem value="role">By Role</SelectItem>
                <SelectItem value="manager_of">Manager Of</SelectItem>
                <SelectItem value="round_robin">Round Robin</SelectItem>
                <SelectItem value="least_loaded">Least Loaded</SelectItem>
              </SelectContent>
            </Select>
            {step.assignee_strategy.type === 'specific_user' && (
              <Input
                value={step.assignee_strategy.user_id}
                onChange={(e) =>
                  onUpdate({
                    assignee_strategy: {
                      type: 'specific_user',
                      user_id: e.target.value,
                    },
                  })
                }
                placeholder="User ID"
                disabled={readOnly}
                className="h-8 text-sm mt-1"
              />
            )}
            {(step.assignee_strategy.type === 'role' ||
              step.assignee_strategy.type === 'least_loaded') && (
              <Input
                value={step.assignee_strategy.role_id}
                onChange={(e) =>
                  onUpdate({
                    assignee_strategy: {
                      ...step.assignee_strategy,
                      role_id: e.target.value,
                    } as AssigneeStrategy,
                  })
                }
                placeholder="Role ID"
                disabled={readOnly}
                className="h-8 text-sm mt-1"
              />
            )}
          </div>
        )}

        {/* ── Timeout ── */}
        <div className="space-y-1.5">
          <Label htmlFor="timeout" className="text-xs">Timeout (minutes)</Label>
          <Input
            id="timeout"
            type="number"
            min={0}
            value={step.timeout_minutes ?? ''}
            onChange={(e) =>
              onUpdate({
                timeout_minutes: e.target.value ? parseInt(e.target.value, 10) : null,
              })
            }
            placeholder="No timeout"
            disabled={readOnly}
            className="h-8 text-sm"
          />
        </div>

        {step.timeout_minutes && (
          <div className="space-y-1.5">
            <Label htmlFor="on-timeout" className="text-xs">On Timeout</Label>
            <Select
              value={step.on_timeout}
              onValueChange={(v) =>
                onUpdate({ on_timeout: v as 'skip' | 'escalate' | 'fail' })
              }
              disabled={readOnly}
            >
              <SelectTrigger id="on-timeout" className="h-8 text-sm">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="skip">Skip</SelectItem>
                <SelectItem value="escalate">Escalate</SelectItem>
                <SelectItem value="fail">Fail</SelectItem>
              </SelectContent>
            </Select>
          </div>
        )}

        {/* Delete button */}
        {!readOnly && (
          <Button
            variant="destructive"
            size="sm"
            className="w-full mt-4"
            onClick={onRemove}
          >
            <Trash2 className="mr-1.5 h-3.5 w-3.5" />
            Remove Step
          </Button>
        )}
      </div>
    </div>
  );
}
