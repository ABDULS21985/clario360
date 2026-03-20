'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { apiGet, apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { timeAgo } from '@/lib/utils';
import { MessageSquare, Send } from 'lucide-react';
import { toast } from 'sonner';
import type { AlertComment } from '@/types/cyber';

interface AlertInvestigationTabProps {
  alertId: string;
}

export function AlertInvestigationTab({ alertId }: AlertInvestigationTabProps) {
  const [comment, setComment] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['alert-comments', alertId],
    queryFn: () =>
      apiGet<{ data: AlertComment[] }>(`${API_ENDPOINTS.CYBER_ALERTS}/${alertId}/comments`),
  });

  const handleSubmit = async () => {
    if (!comment.trim()) return;
    setSubmitting(true);
    try {
      await apiPost(`${API_ENDPOINTS.CYBER_ALERTS}/${alertId}/comment`, { content: comment });
      setComment('');
      void refetch();
      toast.success('Comment added');
    } catch {
      toast.error('Failed to add comment');
    } finally {
      setSubmitting(false);
    }
  };

  const comments = data?.data ?? [];

  return (
    <div className="space-y-4">
      {/* Add comment */}
      <div className="rounded-xl border bg-muted/20 p-4">
        <label className="mb-2 block text-sm font-medium">Add Investigation Note</label>
        <Textarea
          value={comment}
          onChange={(e) => setComment(e.target.value)}
          placeholder="Document your findings, IOC observations, analysis steps…"
          rows={3}
          className="resize-none bg-background"
        />
        <div className="mt-2 flex justify-end">
          <Button
            size="sm"
            onClick={handleSubmit}
            disabled={!comment.trim() || submitting}
          >
            <Send className="mr-1.5 h-3.5 w-3.5" />
            {submitting ? 'Posting…' : 'Post Note'}
          </Button>
        </div>
      </div>

      {/* Comments list */}
      {isLoading ? (
        <LoadingSkeleton variant="list-item" count={4} />
      ) : error ? (
        <ErrorState message="Failed to load comments" onRetry={() => refetch()} />
      ) : comments.length === 0 ? (
        <div className="flex flex-col items-center py-12 text-muted-foreground">
          <MessageSquare className="mb-3 h-8 w-8 opacity-30" />
          <p className="text-sm">No notes yet. Be the first to document your findings.</p>
        </div>
      ) : (
        <div className="space-y-3">
          {comments.map((c) => (
            <div
              key={c.id}
              className={`rounded-xl border p-3.5 ${c.is_system ? 'bg-muted/40 border-dashed' : 'bg-card'}`}
            >
              <div className="mb-1.5 flex items-center justify-between">
                <span className="text-xs font-semibold">
                  {c.is_system ? '🤖 System' : c.user_name}
                </span>
                <span className="text-xs text-muted-foreground">{timeAgo(c.created_at)}</span>
              </div>
              <p className="whitespace-pre-wrap text-sm leading-relaxed">{c.content}</p>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
