'use client';

import { useState } from 'react';
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  ShieldOff,
  Lock,
  Ban,
  Ticket,
  Bell,
  Tag,
  Archive,
  ScanSearch,
  FileCheck,
  CheckSquare,
  Wrench,
  ChevronDown,
  ChevronRight,
} from 'lucide-react';
import type { DSPMRemediationStep } from '@/types/cyber';

interface PlaybookViewerProps {
  playbookId: string;
  steps: DSPMRemediationStep[];
}

const ACTION_ICONS: Record<string, typeof Wrench> = {
  revoke_access: ShieldOff,
  enable_encryption: Lock,
  quarantine_asset: Ban,
  create_ticket: Ticket,
  send_notification: Bell,
  classify_data: Tag,
  archive_asset: Archive,
  scan_asset: ScanSearch,
  update_policy: FileCheck,
  validate_state: CheckSquare,
};

function getActionIcon(action: string) {
  return ACTION_ICONS[action] ?? Wrench;
}

function formatActionLabel(action: string): string {
  return action.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
}

export function PlaybookViewer({ playbookId, steps }: PlaybookViewerProps) {
  const [expandedSteps, setExpandedSteps] = useState<Set<string>>(new Set());
  const sorted = [...steps].sort((a, b) => a.order - b.order);

  const toggleStep = (stepId: string) => {
    setExpandedSteps((prev) => {
      const next = new Set(prev);
      if (next.has(stepId)) {
        next.delete(stepId);
      } else {
        next.add(stepId);
      }
      return next;
    });
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Playbook: {playbookId}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {sorted.map((step) => {
            const Icon = getActionIcon(step.action);
            const hasParams = step.params && Object.keys(step.params).length > 0;
            const isExpanded = expandedSteps.has(step.step_id);

            return (
              <div
                key={step.step_id}
                className="rounded-lg border p-3"
              >
                <div className="flex items-start gap-3">
                  <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-muted text-xs font-semibold">
                    {step.order}
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <Icon className="h-4 w-4 text-muted-foreground" />
                      <Badge variant="secondary" className="text-xs">
                        {formatActionLabel(step.action)}
                      </Badge>
                    </div>
                    <p className="mt-1 text-sm text-muted-foreground">{step.description}</p>

                    {hasParams && (
                      <div className="mt-2">
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          className="h-6 px-2 text-xs"
                          onClick={() => toggleStep(step.step_id)}
                        >
                          {isExpanded ? (
                            <ChevronDown className="mr-1 h-3 w-3" />
                          ) : (
                            <ChevronRight className="mr-1 h-3 w-3" />
                          )}
                          Parameters
                        </Button>

                        {isExpanded && step.params && (
                          <div className="mt-1 rounded bg-muted/50 p-2 text-xs">
                            {Object.entries(step.params).map(([key, value]) => (
                              <div key={key} className="flex items-start gap-2 py-0.5">
                                <span className="font-medium text-muted-foreground">{key}:</span>
                                <span className="break-all">
                                  {typeof value === 'object' ? JSON.stringify(value) : String(value)}
                                </span>
                              </div>
                            ))}
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
}
