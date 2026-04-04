'use client';

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
} from 'lucide-react';
import type { WorkflowStepType } from '@/types/models';

interface PaletteItem {
  type: WorkflowStepType;
  label: string;
  icon: React.ElementType;
}

const groups: { label: string; items: PaletteItem[] }[] = [
  {
    label: 'Human Tasks',
    items: [
      { type: 'approval', label: 'Approval', icon: CheckCircle2 },
      { type: 'review', label: 'Review', icon: Eye },
      { type: 'task', label: 'Task', icon: ClipboardList },
    ],
  },
  {
    label: 'Automation',
    items: [
      { type: 'notification', label: 'Notification', icon: Bell },
      { type: 'webhook', label: 'Webhook', icon: Globe },
      { type: 'script', label: 'Script', icon: Code2 },
      { type: 'sub_workflow', label: 'Sub-workflow', icon: Workflow },
    ],
  },
  {
    label: 'Flow Control',
    items: [
      { type: 'condition', label: 'Condition', icon: GitBranch },
      { type: 'parallel_gateway', label: 'Parallel Fork', icon: GitMerge },
      { type: 'join_gateway', label: 'Parallel Join', icon: GitMerge },
      { type: 'delay', label: 'Delay', icon: Timer },
      { type: 'end', label: 'End', icon: CircleDot },
    ],
  },
];

interface StepPaletteProps {
  onAddStep: (type: WorkflowStepType) => void;
}

export function StepPalette({ onAddStep }: StepPaletteProps) {
  return (
    <div className="w-56 border-r bg-muted/30 overflow-y-auto">
      <div className="p-3 border-b">
        <h3 className="text-sm font-semibold">Step Palette</h3>
        <p className="text-xs text-muted-foreground mt-0.5">
          Drag or click to add steps
        </p>
      </div>
      <div className="p-2 space-y-3">
        {groups.map((group) => (
          <div key={group.label}>
            <h4 className="text-[10px] font-semibold uppercase text-muted-foreground px-2 mb-1">
              {group.label}
            </h4>
            <div className="space-y-0.5">
              {group.items.map((item) => (
                <button
                  key={item.type}
                  className="flex items-center gap-2 w-full rounded-md px-2 py-1.5 text-sm hover:bg-accent transition-colors text-left"
                  draggable
                  onDragStart={(e) => {
                    e.dataTransfer.setData('step-type', item.type);
                    e.dataTransfer.effectAllowed = 'copy';
                  }}
                  onClick={() => onAddStep(item.type)}
                  type="button"
                >
                  <item.icon className="h-4 w-4 shrink-0 text-muted-foreground" />
                  <span>{item.label}</span>
                </button>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
