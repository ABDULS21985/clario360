'use client';

import React, { useCallback, useRef } from 'react';
import {
  CheckCircle2,
  Eye,
  ClipboardList,
  Bell,
  GitBranch,
  GitMerge,
  Timer,
  Globe,
  Code2,
  Workflow,
  CircleDot,
  UserCheck,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import type { WorkflowStep, WorkflowStepType } from '@/types/models';

const stepTypeConfig: Record<
  WorkflowStepType,
  { icon: React.ElementType; color: string; bg: string }
> = {
  approval: { icon: CheckCircle2, color: 'text-blue-700', bg: 'bg-blue-50 border-blue-200' },
  review: { icon: Eye, color: 'text-indigo-700', bg: 'bg-indigo-50 border-indigo-200' },
  task: { icon: ClipboardList, color: 'text-teal-700', bg: 'bg-teal-50 border-teal-200' },
  notification: { icon: Bell, color: 'text-purple-700', bg: 'bg-purple-50 border-purple-200' },
  condition: { icon: GitBranch, color: 'text-yellow-700', bg: 'bg-yellow-50 border-yellow-300' },
  parallel_gateway: { icon: GitMerge, color: 'text-orange-700', bg: 'bg-orange-50 border-orange-200' },
  join_gateway: { icon: GitMerge, color: 'text-orange-700', bg: 'bg-orange-50 border-orange-200' },
  delay: { icon: Timer, color: 'text-gray-700', bg: 'bg-gray-50 border-gray-200' },
  webhook: { icon: Globe, color: 'text-cyan-700', bg: 'bg-cyan-50 border-cyan-200' },
  script: { icon: Code2, color: 'text-green-700', bg: 'bg-green-50 border-green-200' },
  sub_workflow: { icon: Workflow, color: 'text-pink-700', bg: 'bg-pink-50 border-pink-200' },
  end: { icon: CircleDot, color: 'text-red-700', bg: 'bg-red-50 border-red-300' },
};

interface StepNodeProps {
  step: WorkflowStep;
  selected: boolean;
  readOnly: boolean;
  zoom: number;
  stepStatus?: 'completed' | 'running' | 'failed' | 'pending';
  onSelect: (stepId: string) => void;
  onDragStart: (stepId: string, e: React.MouseEvent) => void;
  onConnectStart: (stepId: string, e: React.MouseEvent) => void;
  onConnectEnd: (stepId: string) => void;
}

export const StepNode = React.memo(function StepNode({
  step,
  selected,
  readOnly,
  stepStatus,
  onSelect,
  onDragStart,
  onConnectStart,
  onConnectEnd,
}: StepNodeProps) {
  const config = stepTypeConfig[step.type] ?? stepTypeConfig.task;
  const Icon = config.icon;
  const nodeRef = useRef<HTMLDivElement>(null);

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      if (readOnly) return;
      e.stopPropagation();
      onSelect(step.id);
      onDragStart(step.id, e);
    },
    [readOnly, step.id, onSelect, onDragStart],
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        onSelect(step.id);
      }
    },
    [step.id, onSelect],
  );

  const assigneeLabel = step.assignee_strategy.type === 'specific_user'
    ? 'User'
    : step.assignee_strategy.type === 'role'
      ? 'Role'
      : step.assignee_strategy.type === 'manager_of'
        ? 'Manager'
        : step.assignee_strategy.type === 'round_robin'
          ? 'Round Robin'
          : 'Least Loaded';

  const isHuman = ['approval', 'review', 'task'].includes(step.type);

  return (
    <div
      ref={nodeRef}
      className={cn(
        'absolute select-none rounded-lg border-2 shadow-sm transition-shadow w-[200px]',
        config.bg,
        selected && 'ring-2 ring-primary ring-offset-2 shadow-md',
        stepStatus === 'running' && 'ring-2 ring-blue-500 animate-pulse',
        stepStatus === 'completed' && 'border-green-500 bg-green-50',
        stepStatus === 'failed' && 'border-red-500 bg-red-50',
        stepStatus === 'pending' && 'opacity-50',
      )}
      style={{
        left: step.position.x,
        top: step.position.y,
      }}
      onMouseDown={handleMouseDown}
      tabIndex={0}
      role="button"
      aria-label={`Workflow step: ${step.name}`}
      onKeyDown={handleKeyDown}
    >
      {/* Header */}
      <div className="flex items-center gap-2 px-3 py-2 border-b border-inherit">
        <Icon className={cn('h-4 w-4 shrink-0', config.color)} />
        <span className="text-sm font-medium truncate">{step.name}</span>
      </div>

      {/* Body */}
      <div className="px-3 py-1.5 text-xs text-muted-foreground space-y-0.5">
        <div className="flex items-center justify-between">
          <span className="capitalize">{step.type.replace('_', ' ')}</span>
          {step.timeout_minutes && (
            <span className="text-orange-600">{step.timeout_minutes}m</span>
          )}
        </div>
        {isHuman && (
          <div className="flex items-center gap-1">
            <UserCheck className="h-3 w-3" />
            <span>{assigneeLabel}</span>
          </div>
        )}
      </div>

      {/* Input port (top center) */}
      <div
        className="absolute -top-2 left-1/2 -translate-x-1/2 w-4 h-4 rounded-full bg-white border-2 border-gray-300 hover:border-primary cursor-pointer z-10"
        onMouseUp={(e) => {
          e.stopPropagation();
          onConnectEnd(step.id);
        }}
        aria-label="Input port"
      />

      {/* Output port (bottom center) */}
      {step.type !== 'end' && (
        <div
          className="absolute -bottom-2 left-1/2 -translate-x-1/2 w-4 h-4 rounded-full bg-white border-2 border-gray-300 hover:border-primary cursor-crosshair z-10"
          onMouseDown={(e) => {
            e.stopPropagation();
            onConnectStart(step.id, e);
          }}
          aria-label="Output port"
        />
      )}
    </div>
  );
});
