'use client';

import { useRef, useState } from 'react';
import {
  X,
  CheckCircle,
  XCircle,
  ArrowUpCircle,
  Send,
  UserPlus,
  Loader2,
  ExternalLink,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { StatusBadge } from '@/components/shared/status-badge';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { taskStatusConfig } from '@/lib/status-configs';
import { formatDateTime } from '@/lib/format';
import { formatSLAStatus } from '@/lib/workflow-utils';
import {
  useWorkflowTask,
  useCompleteTask,
  useAssignTask,
  useAddTaskComment,
} from '@/hooks/use-workflow-tasks-ext';
import { TaskFormRenderer } from './task-form-renderer';
import type { HumanTask, TaskComment } from '@/types/models';

interface TaskDetailPanelProps {
  taskId: string;
  onClose: () => void;
}

export function TaskDetailPanel({ taskId, onClose }: TaskDetailPanelProps) {
  const { data: task, isLoading, isError, refetch } = useWorkflowTask(taskId);
  const completeMutation = useCompleteTask();
  const assignMutation = useAssignTask();
  const commentMutation = useAddTaskComment();

  const [commentText, setCommentText] = useState('');
  const [showAssign, setShowAssign] = useState(false);
  const [assignUserId, setAssignUserId] = useState('');
  const formRef = useRef<HTMLFormElement>(null);

  if (isLoading) {
    return (
      <PanelWrapper onClose={onClose}>
        <LoadingSkeleton variant="card" count={2} />
      </PanelWrapper>
    );
  }

  if (isError || !task) {
    return (
      <PanelWrapper onClose={onClose}>
        <ErrorState message="Failed to load task" onRetry={() => refetch()} />
      </PanelWrapper>
    );
  }

  const sla = formatSLAStatus(task);
  const isCompleted = task.status === 'completed' || task.status === 'rejected';
  const isApproval = task.step_id.includes('approval') || task.name.toLowerCase().includes('approv');

  function handleAction(action: 'approve' | 'reject' | 'complete' | 'escalate') {
    if (!task) return;
    // Trigger form submit to validate, then complete
    if (formRef.current && task.form_schema.length > 0 && action !== 'escalate') {
      formRef.current.requestSubmit();
      return;
    }
    completeMutation.mutate({ taskId, data: { action } });
  }

  function handleFormSubmit(formData: Record<string, unknown>) {
    completeMutation.mutate({
      taskId,
      data: { action: isApproval ? 'approve' : 'complete', form_data: formData },
    });
  }

  function handleAssign() {
    if (!assignUserId.trim()) return;
    assignMutation.mutate(
      { taskId, data: { user_id: assignUserId } },
      { onSuccess: () => { setShowAssign(false); setAssignUserId(''); } },
    );
  }

  function handleComment() {
    if (!commentText.trim()) return;
    commentMutation.mutate(
      { taskId, content: commentText },
      { onSuccess: () => setCommentText('') },
    );
  }

  // Comments from metadata (if available)
  const comments: TaskComment[] = (task.metadata?.comments as TaskComment[]) ?? [];

  return (
    <PanelWrapper onClose={onClose}>
      {/* Header */}
      <div className="p-4 border-b space-y-2">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">{task.name}</h2>
          <StatusBadge status={task.status} config={taskStatusConfig} />
        </div>
        <div className="text-xs text-muted-foreground space-y-0.5">
          <p>{task.workflow_name ?? task.definition_name}</p>
          <a
            href={`/workflows/${task.instance_id}`}
            className="inline-flex items-center gap-1 hover:underline text-primary"
          >
            View Instance <ExternalLink className="h-3 w-3" />
          </a>
        </div>
      </div>

      {/* SLA & Assignee */}
      <div className="p-4 border-b space-y-2">
        {task.sla_deadline && (
          <div className="flex justify-between text-sm">
            <span className="text-muted-foreground">Due</span>
            <span className={sla.color}>
              {sla.text}
            </span>
          </div>
        )}
        <div className="flex justify-between text-sm">
          <span className="text-muted-foreground">Assigned to</span>
          <span>{task.claimed_by_name ?? task.assignee_id ?? 'Unassigned'}</span>
        </div>
        {!isCompleted && (
          <Button
            variant="outline"
            size="sm"
            className="w-full"
            onClick={() => setShowAssign(!showAssign)}
          >
            <UserPlus className="mr-1 h-3.5 w-3.5" />
            Reassign
          </Button>
        )}
        {showAssign && (
          <div className="flex gap-2">
            <Input
              value={assignUserId}
              onChange={(e) => setAssignUserId(e.target.value)}
              placeholder="User ID"
              className="h-8 text-sm"
            />
            <Button
              size="sm"
              onClick={handleAssign}
              disabled={assignMutation.isPending}
            >
              {assignMutation.isPending ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                'Assign'
              )}
            </Button>
          </div>
        )}
      </div>

      {/* Form */}
      {task.form_schema.length > 0 && (
        <div className="p-4 border-b">
          <h3 className="text-sm font-medium mb-2">Form</h3>
          <TaskFormRenderer
            fields={task.form_schema}
            initialData={task.form_data}
            readOnly={isCompleted}
            onSubmit={handleFormSubmit}
            formRef={formRef}
          />
        </div>
      )}

      {/* Action buttons */}
      {!isCompleted && (
        <div className="p-4 border-b flex flex-wrap gap-2">
          {isApproval ? (
            <>
              <Button
                size="sm"
                onClick={() => handleAction('approve')}
                disabled={completeMutation.isPending}
              >
                {completeMutation.isPending ? (
                  <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" />
                ) : (
                  <CheckCircle className="mr-1 h-3.5 w-3.5" />
                )}
                Approve
              </Button>
              <Button
                variant="destructive"
                size="sm"
                onClick={() => handleAction('reject')}
                disabled={completeMutation.isPending}
              >
                <XCircle className="mr-1 h-3.5 w-3.5" />
                Reject
              </Button>
            </>
          ) : (
            <Button
              size="sm"
              onClick={() => handleAction('complete')}
              disabled={completeMutation.isPending}
            >
              {completeMutation.isPending ? (
                <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" />
              ) : (
                <CheckCircle className="mr-1 h-3.5 w-3.5" />
              )}
              Complete
            </Button>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={() => handleAction('escalate')}
            disabled={completeMutation.isPending}
          >
            <ArrowUpCircle className="mr-1 h-3.5 w-3.5" />
            Escalate
          </Button>
        </div>
      )}

      {/* Comments */}
      <div className="p-4 space-y-3">
        <h3 className="text-sm font-medium">Comments</h3>
        {comments.length === 0 && (
          <p className="text-xs text-muted-foreground">No comments yet.</p>
        )}
        <div className="space-y-2 max-h-48 overflow-y-auto">
          {comments.map((c) => (
            <div key={c.id} className="text-xs border rounded p-2">
              <div className="flex justify-between text-muted-foreground mb-0.5">
                <span className="font-medium">{c.user_name}</span>
                <span>{formatDateTime(c.created_at)}</span>
              </div>
              <p>{c.content}</p>
            </div>
          ))}
        </div>
        <div className="flex gap-2">
          <Textarea
            value={commentText}
            onChange={(e) => setCommentText(e.target.value)}
            placeholder="Add a comment..."
            rows={2}
            className="text-sm"
          />
          <Button
            size="icon"
            className="shrink-0 self-end"
            onClick={handleComment}
            disabled={!commentText.trim() || commentMutation.isPending}
          >
            {commentMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Send className="h-4 w-4" />
            )}
          </Button>
        </div>
      </div>

      {/* Instance context */}
      {task.metadata && Object.keys(task.metadata).length > 0 && (
        <details className="p-4 border-t">
          <summary className="text-xs text-muted-foreground cursor-pointer hover:text-foreground">
            Instance Context
          </summary>
          <pre className="mt-2 text-xs bg-muted rounded p-2 overflow-x-auto">
            {JSON.stringify(task.metadata, null, 2)}
          </pre>
        </details>
      )}
    </PanelWrapper>
  );
}

function PanelWrapper({
  children,
  onClose,
}: {
  children: React.ReactNode;
  onClose: () => void;
}) {
  return (
    <div className="fixed inset-y-0 right-0 w-[420px] max-w-full bg-background border-l shadow-xl z-50 flex flex-col overflow-y-auto">
      <div className="flex items-center justify-between px-4 py-2 border-b bg-muted/30">
        <h3 className="text-sm font-semibold">Task Details</h3>
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7"
          onClick={onClose}
          aria-label="Close panel"
        >
          <X className="h-4 w-4" />
        </Button>
      </div>
      <div className="flex-1 overflow-y-auto">{children}</div>
    </div>
  );
}
