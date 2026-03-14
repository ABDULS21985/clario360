'use client';

import { useMemo, useState } from 'react';
import { ArrowUpCircle, CheckCircle2, Search, ShieldAlert, UserCheck } from 'lucide-react';
import { PermissionGate } from '@/components/auth/permission-gate';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { Button } from '@/components/ui/button';
import { apiPut } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { ALERT_STATUS_TRANSITIONS } from '@/lib/cyber-alerts';
import type { AlertStatus, CyberAlert } from '@/types/cyber';

import { AlertAssignDialog } from '../../_components/alert-assign-dialog';
import { AlertEscalateDialog } from '../../_components/alert-escalate-dialog';
import { AlertFalsePositiveDialog } from '../../_components/alert-false-positive-dialog';
import { AlertStatusDialog } from '../../_components/alert-status-dialog';

interface AlertActionsProps {
  alert: CyberAlert;
  onUpdated: () => void;
}

export function AlertActions({ alert, onUpdated }: AlertActionsProps) {
  const [assignOpen, setAssignOpen] = useState(false);
  const [escalateOpen, setEscalateOpen] = useState(false);
  const [falsePositiveOpen, setFalsePositiveOpen] = useState(false);
  const [statusDialogOpen, setStatusDialogOpen] = useState(false);
  const [statusDialogTarget, setStatusDialogTarget] = useState<AlertStatus | undefined>(undefined);
  const [confirmStatus, setConfirmStatus] = useState<AlertStatus | null>(null);

  const allowed = useMemo(
    () => new Set(ALERT_STATUS_TRANSITIONS[alert.status] ?? []),
    [alert.status],
  );

  async function handleConfirmTransition() {
    if (!confirmStatus) {
      return;
    }
    await apiPut(API_ENDPOINTS.CYBER_ALERT_STATUS(alert.id), { status: confirmStatus });
    setConfirmStatus(null);
    onUpdated();
  }

  const investigationLabel = alert.status === 'acknowledged' ? 'Start Investigation' : 'Reopen';

  return (
    <PermissionGate
      permission="cyber:write"
      fallback={(
        <p className="text-sm text-muted-foreground">
          You have read-only access to this alert.
        </p>
      )}
    >
      <div className="flex flex-wrap items-center gap-2">
        {allowed.has('acknowledged') && (
          <Button size="sm" onClick={() => setConfirmStatus('acknowledged')}>
            <CheckCircle2 className="mr-1.5 h-4 w-4" />
            Acknowledge
          </Button>
        )}

        {allowed.has('investigating') && (
          <Button variant="outline" size="sm" onClick={() => setConfirmStatus('investigating')}>
            <Search className="mr-1.5 h-4 w-4" />
            {investigationLabel}
          </Button>
        )}

        {allowed.has('resolved') && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              setStatusDialogTarget('resolved');
              setStatusDialogOpen(true);
            }}
          >
            Resolve
          </Button>
        )}

        {allowed.has('closed') && (
          <Button variant="outline" size="sm" onClick={() => setConfirmStatus('closed')}>
            Close
          </Button>
        )}

        {allowed.has('escalated') && (
          <Button variant="outline" size="sm" onClick={() => setEscalateOpen(true)}>
            <ArrowUpCircle className="mr-1.5 h-4 w-4" />
            Escalate
          </Button>
        )}

        {allowed.has('false_positive') && (
          <Button variant="outline" size="sm" onClick={() => setFalsePositiveOpen(true)}>
            <ShieldAlert className="mr-1.5 h-4 w-4" />
            Mark False Positive
          </Button>
        )}

        <Button variant="outline" size="sm" onClick={() => setAssignOpen(true)}>
          <UserCheck className="mr-1.5 h-4 w-4" />
          Assign
        </Button>

        <Button
          variant="ghost"
          size="sm"
          onClick={() => {
            setStatusDialogTarget(undefined);
            setStatusDialogOpen(true);
          }}
        >
          Change Status
        </Button>
      </div>

      <AlertAssignDialog
        open={assignOpen}
        onOpenChange={setAssignOpen}
        alert={alert}
        onSuccess={onUpdated}
      />

      <AlertEscalateDialog
        open={escalateOpen}
        onOpenChange={setEscalateOpen}
        alert={alert}
        onSuccess={onUpdated}
      />

      <AlertFalsePositiveDialog
        open={falsePositiveOpen}
        onOpenChange={setFalsePositiveOpen}
        alert={alert}
        onSuccess={onUpdated}
      />

      <AlertStatusDialog
        open={statusDialogOpen}
        onOpenChange={setStatusDialogOpen}
        alert={alert}
        initialStatus={statusDialogTarget}
        onSuccess={onUpdated}
      />

      <ConfirmDialog
        open={confirmStatus !== null}
        onOpenChange={(open) => {
          if (!open) {
            setConfirmStatus(null);
          }
        }}
        title={confirmTitle(confirmStatus)}
        description={confirmDescription(alert, confirmStatus)}
        confirmLabel={confirmLabel(confirmStatus)}
        onConfirm={handleConfirmTransition}
      />
    </PermissionGate>
  );
}

function confirmTitle(status: AlertStatus | null): string {
  switch (status) {
    case 'acknowledged':
      return 'Acknowledge Alert';
    case 'investigating':
      return 'Move To Investigation';
    case 'closed':
      return 'Close Alert';
    default:
      return 'Confirm Status Change';
  }
}

function confirmLabel(status: AlertStatus | null): string {
  switch (status) {
    case 'acknowledged':
      return 'Acknowledge';
    case 'investigating':
      return 'Start Investigation';
    case 'closed':
      return 'Close Alert';
    default:
      return 'Continue';
  }
}

function confirmDescription(alert: CyberAlert, status: AlertStatus | null): string {
  switch (status) {
    case 'acknowledged':
      return `This will acknowledge ${alert.title} and auto-assign it to you if it is still unowned.`;
    case 'investigating':
      return `This will move ${alert.title} into the investigating state so analysts can continue the case.`;
    case 'closed':
      return `This will close ${alert.title} and end the active investigation workflow.`;
    default:
      return `Update ${alert.title}.`;
  }
}
