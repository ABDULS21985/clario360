'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { MessageSquareText, Send, Sparkles } from 'lucide-react';
import { toast } from 'sonner';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { Avatar, AvatarFallback } from '@/components/ui/avatar';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { apiGet, apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { getAvatarColor, getInitials } from '@/lib/format';
import { timeAgo } from '@/lib/utils';
import type { AlertComment } from '@/types/cyber';

interface AlertCommentsProps {
  alertId: string;
}

export function AlertComments({ alertId }: AlertCommentsProps) {
  const [draft, setDraft] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const commentsQuery = useQuery({
    queryKey: ['alert-comments', alertId],
    queryFn: () => apiGet<{ data: AlertComment[] }>(API_ENDPOINTS.CYBER_ALERT_COMMENTS(alertId)),
  });

  async function handleSubmit() {
    if (!draft.trim()) {
      return;
    }

    setSubmitting(true);
    try {
      await apiPost(API_ENDPOINTS.CYBER_ALERT_COMMENTS(alertId), {
        content: draft.trim(),
      });
      setDraft('');
      toast.success('Comment added');
      await commentsQuery.refetch();
    } catch {
      toast.error('Failed to add comment');
    } finally {
      setSubmitting(false);
    }
  }

  const comments = commentsQuery.data?.data ?? [];

  return (
    <div className="space-y-6">
      <section className="rounded-[26px] border bg-card p-5 shadow-sm">
        <div className="flex items-start gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-2xl border border-slate-200 bg-slate-50 text-slate-700">
            <MessageSquareText className="h-4 w-4" />
          </div>
          <div className="flex-1 space-y-3">
            <div>
              <p className="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Investigation Comments
              </p>
              <h2 className="text-lg font-semibold tracking-[-0.03em] text-slate-950">
                Analyst Collaboration
              </h2>
            </div>
            <Textarea
              value={draft}
              onChange={(event) => setDraft(event.target.value)}
              rows={5}
              placeholder="Document findings, note pivots, or mention a teammate with @name."
            />
            <div className="flex justify-end">
              <Button onClick={() => void handleSubmit()} disabled={submitting || !draft.trim()}>
                <Send className="mr-2 h-4 w-4" />
                {submitting ? 'Posting…' : 'Add Comment'}
              </Button>
            </div>
          </div>
        </div>
      </section>

      <section className="space-y-3">
        {commentsQuery.isLoading ? (
          <LoadingSkeleton variant="list-item" count={4} />
        ) : commentsQuery.error ? (
          <ErrorState message="Failed to load comments" onRetry={() => void commentsQuery.refetch()} />
        ) : comments.length === 0 ? (
          <div className="rounded-[26px] border border-dashed bg-card p-8 text-center text-muted-foreground">
            <Sparkles className="mx-auto mb-3 h-6 w-6 opacity-60" />
            No investigation comments yet.
          </div>
        ) : (
          comments.map((comment) => <CommentCard key={comment.id} comment={comment} />)
        )}
      </section>
    </div>
  );
}

function CommentCard({ comment }: { comment: AlertComment }) {
  const [firstName, ...rest] = comment.user_name.split(' ');
  const lastName = rest.join(' ');
  const initials = getInitials(firstName || '?', lastName || '');
  const tone = getAvatarColor(comment.user_name || comment.user_email || 'system');
  const mentions = Array.isArray(comment.metadata?.['mentions'])
    ? (comment.metadata?.['mentions'] as string[])
    : [];

  return (
    <article className="rounded-[26px] border bg-card p-5 shadow-sm">
      <div className="flex items-start gap-4">
        <Avatar className="h-11 w-11">
          <AvatarFallback className={`${tone} text-sm font-semibold text-white`}>
            {comment.is_system ? 'AI' : initials}
          </AvatarFallback>
        </Avatar>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <p className="text-sm font-semibold text-slate-950">
              {comment.is_system ? 'System' : comment.user_name}
            </p>
            <span className="text-xs text-muted-foreground">{timeAgo(comment.created_at)}</span>
            {comment.user_email && !comment.is_system && (
              <span className="text-xs text-muted-foreground">{comment.user_email}</span>
            )}
          </div>
          <p className="mt-3 whitespace-pre-wrap text-sm leading-7 text-slate-700">{comment.content}</p>
          {mentions.length > 0 && (
            <div className="mt-3 flex flex-wrap gap-2">
              {mentions.map((mention) => (
                <span
                  key={mention}
                  className="rounded-full bg-emerald-50 px-3 py-1 text-xs font-medium text-emerald-700"
                >
                  @{mention}
                </span>
              ))}
            </div>
          )}
        </div>
      </div>
    </article>
  );
}
