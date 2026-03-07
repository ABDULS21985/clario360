'use client';

import { useState, useEffect, useRef } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { ArrowLeft, X, AlertCircle } from 'lucide-react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { apiGet } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { StatusBadge } from '@/components/shared/status-badge';
import {
  TaskDetailForm,
  type TaskDetailFormHandle,
} from '@/components/workflows/task-detail-form';
import { TaskContextPanel } from '@/components/workflows/task-context-panel';
import { TaskClaimButton } from '@/components/workflows/task-claim-button';
import { TaskCompleteDialog } from '@/components/workflows/task-complete-dialog';
import { TaskRejectDialog } from '@/components/workflows/task-reject-dialog';
import { TaskDelegateDialog } from '@/components/workflows/task-delegate-dialog';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { taskStatusConfig } from '@/lib/status-configs';
import {
  canClaimTask,
  canDelegateTask,
  formatSLAStatus,
  PRIORITY_LABELS,
} from '@/lib/workflow-utils';
import { formatDateTime } from '@/lib/utils';
import { useAuth } from '@/hooks/use-auth';
import type { HumanTask } from '@/types/models';

const DRAFT_KEY = (id: string) => `clario360_task_draft_${id}`;
const FORM_ID = 'task-detail-form';

interface DraftData {
  values: Record<string, unknown>;
  savedAt: string;
}

export default function TaskDetailPage() {
  const params = useParams();
  const router = useRouter();
  const queryClient = useQueryClient();
  const taskId = params.id as string;
  const { user } = useAuth();
  const formRef = useRef<TaskDetailFormHandle>(null);

  const [formData, setFormData] = useState<Record<string, unknown>>({});
  const [draftData, setDraftData] = useState<DraftData | null>(null);
  const [showDraftBanner, setShowDraftBanner] = useState(false);
  const [completeOpen, setCompleteOpen] = useState(false);
  const [rejectOpen, setRejectOpen] = useState(false);
  const [delegateOpen, setDelegateOpen] = useState(false);

  const {
    data: task,
    isLoading,
    isError,
    refetch,
  } = useQuery({
    queryKey: ['task', taskId],
    queryFn: () => apiGet<HumanTask>(`/api/v1/workflows/tasks/${taskId}`),
  });

  useEffect(() => {
    if (!taskId || !task || task.status === 'completed' || task.status === 'cancelled') {
      return;
    }

    const raw = localStorage.getItem(DRAFT_KEY(taskId));
    if (!raw) {
      return;
    }

    try {
      const parsed = JSON.parse(raw) as DraftData;
      setDraftData(parsed);
      setShowDraftBanner(true);
    } catch {
      localStorage.removeItem(DRAFT_KEY(taskId));
    }
  }, [task, taskId]);

  const saveDraft = (values: Record<string, unknown>) => {
    const draft: DraftData = { values, savedAt: new Date().toISOString() };
    localStorage.setItem(DRAFT_KEY(taskId), JSON.stringify(draft));
    setDraftData(draft);
    setShowDraftBanner(true);
  };

  const discardDraft = () => {
    localStorage.removeItem(DRAFT_KEY(taskId));
    setDraftData(null);
    setShowDraftBanner(false);
    setFormData({});
  };

  const handleFormSubmit = async (data: Record<string, unknown>) => {
    setFormData(data);
    setCompleteOpen(true);
  };

  const handleCompleteSuccess = () => {
    localStorage.removeItem(DRAFT_KEY(taskId));
    router.push('/workflows/tasks');
  };

  const handleRejectSuccess = () => {
    router.push('/workflows/tasks');
  };

  const handleDelegateSuccess = () => {
    router.push('/workflows/tasks');
  };

  const handleClaimSuccess = () => {
    queryClient.invalidateQueries({ queryKey: ['task', taskId] });
  };

  if (isLoading) {
    return (
      <div className="space-y-6">
        <LoadingSkeleton variant="card" count={3} />
      </div>
    );
  }

  if (isError || !task) {
    return (
      <ErrorState
        message="Task not found or failed to load."
        onRetry={() => refetch()}
      />
    );
  }

  const isCompleted = task.status === 'completed' || task.status === 'cancelled';
  const isReadOnly = isCompleted || task.status === 'rejected';
  const isMine = task.claimed_by === user?.id;
  const isUnclaimed = !task.claimed_by && task.status === 'pending';
  const isClaimedByOther = Boolean(task.claimed_by && task.claimed_by !== user?.id);
  const canClaim = canClaimTask(task, user);
  const canDelegate = canDelegateTask(task, user);
  const sla = formatSLAStatus(task);
  const priorityLabel = PRIORITY_LABELS[task.priority] ?? 'Normal';
  const hasForm = task.form_schema.length > 0;
  const currentInitialValues =
    draftData?.values ?? (isCompleted && task.form_data ? task.form_data : task.form_data ?? {});

  return (
    <div className="space-y-6">
      <button
        onClick={() => router.push('/workflows/tasks')}
        className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
        type="button"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to My Tasks
      </button>

      <div className="space-y-2">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h1 className="text-xl font-bold">{task.name}</h1>
            <p className="text-sm text-muted-foreground">
              {task.definition_name || task.workflow_name}
              {task.step_id && ` · ${task.step_id}`}
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant="outline" className="text-xs">
              {priorityLabel}
            </Badge>
            <StatusBadge status={task.status} config={taskStatusConfig} />
            <span className={`text-xs font-medium ${sla.color}`}>{sla.text}</span>
          </div>
        </div>
        {task.description && <p className="text-sm text-muted-foreground">{task.description}</p>}
      </div>

      {showDraftBanner && draftData && (
        <div className="flex items-center gap-3 rounded-lg border border-blue-200 bg-blue-50 px-4 py-2 text-sm">
          <AlertCircle className="h-4 w-4 text-blue-600" />
          <span className="text-blue-800">
            Draft restored from {formatDateTime(draftData.savedAt)}.
          </span>
          <button
            onClick={discardDraft}
            className="ml-auto flex items-center gap-1 text-xs text-blue-700 hover:underline"
            type="button"
          >
            <X className="h-3 w-3" />
            Discard draft
          </button>
        </div>
      )}

      {isClaimedByOther && !isCompleted && (
        <div className="rounded-lg border border-orange-200 bg-orange-50 px-4 py-2 text-sm text-orange-800">
          This task is claimed by{' '}
          <span className="font-medium">{task.claimed_by_name ?? task.claimed_by}</span>.
          You are viewing in read-only mode.
        </div>
      )}

      {isCompleted && (
        <div className="rounded-lg border border-green-200 bg-green-50 px-4 py-2 text-sm text-green-800">
          This task has been <strong>{task.status}</strong>. Showing submitted data.
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-[3fr_2fr]">
        <div className="space-y-4">
          {isUnclaimed ? (
            canClaim ? (
              <TaskClaimButton task={task} onSuccess={handleClaimSuccess} />
            ) : (
              <div className="rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">
                This task is currently unclaimed and restricted to the required role.
              </div>
            )
          ) : hasForm ? (
            <div className="rounded-lg border p-6">
              <h2 className="mb-4 text-base font-semibold">Task Form</h2>
              <TaskDetailForm
                ref={formRef}
                formId={FORM_ID}
                showSubmitButton={false}
                formSchema={task.form_schema}
                initialValues={currentInitialValues}
                readOnly={isReadOnly || isClaimedByOther}
                onSubmit={handleFormSubmit}
                onDraftSave={isMine ? saveDraft : undefined}
              />
            </div>
          ) : (
            <div className="rounded-lg border p-6 text-center text-sm text-muted-foreground">
              No form fields required for this task.
            </div>
          )}
        </div>

        <div>
          <TaskContextPanel task={task} instanceId={task.instance_id} />
        </div>
      </div>

      {!isCompleted && !isUnclaimed && (
        <div className="flex flex-wrap items-center justify-between gap-3 border-t pt-4">
          <div className="flex gap-2">
            {isMine && (
              <Button variant="outline" onClick={() => setRejectOpen(true)}>
                Reject
              </Button>
            )}
            {canDelegate && (
              <Button variant="outline" onClick={() => setDelegateOpen(true)}>
                Delegate
              </Button>
            )}
          </div>
          {isMine && (
            <div className="flex gap-2">
              {hasForm && (
                <Button
                  variant="outline"
                  onClick={() => formRef.current?.saveDraft()}
                >
                  Save Draft
                </Button>
              )}
              <Button
                onClick={() => {
                  if (hasForm) {
                    void formRef.current?.submit();
                  } else {
                    setFormData({});
                    setCompleteOpen(true);
                  }
                }}
              >
                Complete ✓
              </Button>
            </div>
          )}
        </div>
      )}

      <TaskCompleteDialog
        task={task}
        formData={formData}
        open={completeOpen}
        onOpenChange={setCompleteOpen}
        onSuccess={handleCompleteSuccess}
      />

      <TaskRejectDialog
        task={task}
        open={rejectOpen}
        onOpenChange={setRejectOpen}
        onSuccess={handleRejectSuccess}
      />

      <TaskDelegateDialog
        task={task}
        open={delegateOpen}
        onOpenChange={setDelegateOpen}
        onSuccess={handleDelegateSuccess}
      />
    </div>
  );
}
