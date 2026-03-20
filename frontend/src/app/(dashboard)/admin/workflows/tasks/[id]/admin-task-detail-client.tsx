'use client';

import { useRef, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import {
  ArrowLeft,
  CheckCircle,
  XCircle,
  ArrowUpCircle,
  Send,
  UserPlus,
  Loader2,
  ExternalLink,
  MessageSquare,
  Clock,
  User,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { StatusBadge } from '@/components/shared/status-badge';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { taskStatusConfig } from '@/lib/status-configs';
import { formatDateTime } from '@/lib/format';
import { formatSLAStatus, PRIORITY_LABELS } from '@/lib/workflow-utils';
import {
  useWorkflowTask,
  useCompleteTask,
  useAssignTask,
  useAddTaskComment,
} from '@/hooks/use-workflow-tasks-ext';
import { TaskFormRenderer } from '../components/task-form-renderer';
import type { TaskComment } from '@/types/models';

export function AdminTaskDetailClient() {
  const params = useParams();
  const router = useRouter();
  const taskId = (params?.id as string | undefined) ?? '';
  const formRef = useRef<HTMLFormElement>(null);

  const [commentText, setCommentText] = useState('');
  const [showAssign, setShowAssign] = useState(false);
  const [assignUserId, setAssignUserId] = useState('');

  const { data: task, isLoading, isError, refetch } = useWorkflowTask(taskId);
  const completeMutation = useCompleteTask();
  const assignMutation = useAssignTask();
  const commentMutation = useAddTaskComment();

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
        message="Failed to load task"
        onRetry={() => refetch()}
      />
    );
  }

  const sla = formatSLAStatus(task);
  const isCompleted = task.status === 'completed' || task.status === 'rejected' || task.status === 'cancelled';
  const isApproval = task.step_id.includes('approval') || task.name.toLowerCase().includes('approv');
  const priorityLabel = PRIORITY_LABELS[task.priority] ?? 'Normal';
  const comments: TaskComment[] = (task.metadata?.comments as TaskComment[] | undefined) ?? [];

  function handleAction(action: 'approve' | 'reject' | 'complete' | 'escalate') {
    if (formRef.current && task!.form_schema.length > 0 && action !== 'escalate') {
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

  return (
    <div className="space-y-6">
      {/* Back */}
      <button
        onClick={() => router.push('/admin/workflows/tasks')}
        className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
        type="button"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to Task Queue
      </button>

      {/* Header */}
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-xl font-bold">{task.name}</h1>
          <div className="mt-1.5 flex flex-wrap items-center gap-2">
            <StatusBadge status={task.status} config={taskStatusConfig} />
            <Badge variant="outline" className="text-xs">{priorityLabel}</Badge>
            {task.definition_name && (
              <span className="text-sm text-muted-foreground">
                {task.definition_name}
              </span>
            )}
          </div>
          {task.description && (
            <p className="mt-1 text-sm text-muted-foreground">{task.description}</p>
          )}
        </div>

        <a
          href={`/admin/workflows/instances/${task.instance_id}`}
          className="flex items-center gap-1 text-sm text-primary hover:underline"
        >
          View Instance <ExternalLink className="h-3.5 w-3.5" />
        </a>
      </div>

      {/* Main 2-col layout */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-[3fr_2fr]">
        {/* Left: Form + Actions */}
        <div className="space-y-4">
          {/* Form */}
          {task.form_schema.length > 0 ? (
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">Task Form</CardTitle>
              </CardHeader>
              <CardContent>
                <TaskFormRenderer
                  fields={task.form_schema}
                  initialData={task.form_data}
                  readOnly={isCompleted}
                  onSubmit={handleFormSubmit}
                  formRef={formRef}
                />
              </CardContent>
            </Card>
          ) : (
            <div className="rounded-lg border border-dashed p-6 text-center text-sm text-muted-foreground">
              No form fields required for this task.
            </div>
          )}

          {/* Action buttons */}
          {!isCompleted && (
            <div className="flex flex-wrap gap-2 border-t pt-4">
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
        </div>

        {/* Right: Context + Metadata */}
        <div className="space-y-4">
          {/* SLA & Assignee card */}
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium">Task Info</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {task.sla_deadline && (
                <div className="flex items-center justify-between text-sm">
                  <span className="flex items-center gap-1.5 text-muted-foreground">
                    <Clock className="h-3.5 w-3.5" />
                    Due
                  </span>
                  <span className={sla.color}>{sla.text}</span>
                </div>
              )}
              <div className="flex items-center justify-between text-sm">
                <span className="flex items-center gap-1.5 text-muted-foreground">
                  <User className="h-3.5 w-3.5" />
                  Assigned to
                </span>
                <span>{task.claimed_by_name ?? task.assignee_id ?? 'Unassigned'}</span>
              </div>
              {task.assignee_role && (
                <div className="flex items-center justify-between text-sm">
                  <span className="text-muted-foreground">Required Role</span>
                  <Badge variant="secondary" className="text-xs">
                    {task.assignee_role}
                  </Badge>
                </div>
              )}
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Created</span>
                <span>{formatDateTime(task.created_at)}</span>
              </div>
              {task.claimed_at && (
                <div className="flex items-center justify-between text-sm">
                  <span className="text-muted-foreground">Claimed</span>
                  <span>{formatDateTime(task.claimed_at)}</span>
                </div>
              )}

              {!isCompleted && (
                <div className="pt-1">
                  <Button
                    variant="outline"
                    size="sm"
                    className="w-full"
                    onClick={() => setShowAssign(!showAssign)}
                  >
                    <UserPlus className="mr-1 h-3.5 w-3.5" />
                    Reassign
                  </Button>
                  {showAssign && (
                    <div className="mt-2 flex gap-2">
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
              )}
            </CardContent>
          </Card>

          {/* Instance context */}
          {task.metadata && Object.keys(task.metadata).length > 0 && (
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">Instance Context</CardTitle>
              </CardHeader>
              <CardContent>
                <pre className="max-h-40 overflow-x-auto overflow-y-auto rounded bg-muted p-2 text-xs">
                  {JSON.stringify(task.metadata, null, 2)}
                </pre>
              </CardContent>
            </Card>
          )}
        </div>
      </div>

      {/* Comments */}
      <div className="rounded-lg border p-4">
        <div className="mb-3 flex items-center gap-2">
          <MessageSquare className="h-4 w-4 text-muted-foreground" />
          <h3 className="text-sm font-semibold">
            Comments {comments.length > 0 && `(${comments.length})`}
          </h3>
        </div>
        {comments.length === 0 ? (
          <p className="text-xs text-muted-foreground">No comments yet.</p>
        ) : (
          <div className="mb-3 space-y-3">
            {comments.map((c) => (
              <div key={c.id} className="rounded border p-2 text-xs">
                <div className="mb-0.5 flex justify-between text-muted-foreground">
                  <span className="font-medium">{c.user_name}</span>
                  <span>{formatDateTime(c.created_at)}</span>
                </div>
                <p>{c.content}</p>
              </div>
            ))}
          </div>
        )}
        <div className="mt-3 flex gap-2">
          <Textarea
            value={commentText}
            onChange={(e) => setCommentText(e.target.value)}
            placeholder="Add a comment..."
            rows={2}
            className="text-sm"
          />
          <Button
            size="icon"
            variant="outline"
            className="shrink-0 self-end"
            disabled={!commentText.trim() || commentMutation.isPending}
            onClick={handleComment}
          >
            {commentMutation.isPending ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <Send className="h-3.5 w-3.5" />
            )}
          </Button>
        </div>
      </div>
    </div>
  );
}
