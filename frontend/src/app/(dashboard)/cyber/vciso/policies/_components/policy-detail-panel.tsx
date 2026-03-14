'use client';

import { useState } from 'react';
import {
  Calendar,
  Clock,
  Edit,
  Send,
  CheckCircle,
  Archive,
  User,
  Tag,
} from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { DetailPanel } from '@/components/shared/detail-panel';
import { StatusBadge } from '@/components/shared/status-badge';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { policyStatusConfig } from '@/lib/status-configs';
import { formatDate } from '@/lib/format';
import { titleCase } from '@/lib/format';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import { cn } from '@/lib/utils';
import type { VCISOPolicy, PolicyStatus } from '@/types/cyber';

interface PolicyDetailPanelProps {
  policy: VCISOPolicy;
  open: boolean;
  onClose: () => void;
  onEdit: () => void;
  onRefresh: () => void;
}

export function PolicyDetailPanel({
  policy,
  open,
  onClose,
  onEdit,
  onRefresh,
}: PolicyDetailPanelProps) {
  const [confirmAction, setConfirmAction] = useState<{
    type: 'submit_review' | 'publish' | 'retire';
    title: string;
    description: string;
  } | null>(null);

  const statusMutation = useApiMutation<VCISOPolicy, { status: PolicyStatus }>(
    'put',
    () => `${API_ENDPOINTS.CYBER_VCISO_POLICIES}/${policy.id}/status`,
    {
      invalidateKeys: ['vciso-policies'],
      onSuccess: () => {
        onRefresh();
        setConfirmAction(null);
      },
    },
  );

  const handleStatusChange = async () => {
    if (!confirmAction) return;

    const statusMap: Record<string, PolicyStatus> = {
      submit_review: 'review',
      publish: 'published',
      retire: 'retired',
    };

    statusMutation.mutate({ status: statusMap[confirmAction.type] });
  };

  const isOverdue = new Date(policy.review_due) < new Date();

  return (
    <>
      <DetailPanel
        open={open}
        onOpenChange={(o) => !o && onClose()}
        title={policy.title}
        description={`Version ${policy.version}`}
        width="xl"
      >
        <div className="space-y-6">
          {/* Status and Actions Bar */}
          <div className="flex items-center justify-between">
            <StatusBadge
              status={policy.status}
              config={policyStatusConfig}
              size="lg"
            />
            <div className="flex items-center gap-2">
              {(policy.status === 'draft' || policy.status === 'approved') && (
                <Button variant="outline" size="sm" onClick={onEdit}>
                  <Edit className="mr-1.5 h-3.5 w-3.5" />
                  Edit
                </Button>
              )}
              {policy.status === 'draft' && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() =>
                    setConfirmAction({
                      type: 'submit_review',
                      title: 'Submit for Review',
                      description:
                        'This will submit the policy for review. Reviewers will be notified.',
                    })
                  }
                >
                  <Send className="mr-1.5 h-3.5 w-3.5" />
                  Submit for Review
                </Button>
              )}
              {policy.status === 'approved' && (
                <Button
                  size="sm"
                  onClick={() =>
                    setConfirmAction({
                      type: 'publish',
                      title: 'Publish Policy',
                      description:
                        'This will publish the policy and make it active across the organization.',
                    })
                  }
                >
                  <CheckCircle className="mr-1.5 h-3.5 w-3.5" />
                  Publish
                </Button>
              )}
              {policy.status === 'published' && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() =>
                    setConfirmAction({
                      type: 'retire',
                      title: 'Retire Policy',
                      description:
                        'This will retire the policy. It will no longer be active but will remain in the archive.',
                    })
                  }
                >
                  <Archive className="mr-1.5 h-3.5 w-3.5" />
                  Retire
                </Button>
              )}
            </div>
          </div>

          <Separator />

          {/* Metadata Grid */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1">
              <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                Domain
              </p>
              <Badge variant="outline">{titleCase(policy.domain)}</Badge>
            </div>

            <div className="space-y-1">
              <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                Version
              </p>
              <p className="text-sm font-medium">{policy.version}</p>
            </div>

            <div className="space-y-1">
              <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                Owner
              </p>
              <div className="flex items-center gap-1.5">
                <User className="h-3.5 w-3.5 text-muted-foreground" />
                <p className="text-sm">{policy.owner_name}</p>
              </div>
            </div>

            {policy.reviewer_name && (
              <div className="space-y-1">
                <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  Reviewer
                </p>
                <div className="flex items-center gap-1.5">
                  <User className="h-3.5 w-3.5 text-muted-foreground" />
                  <p className="text-sm">{policy.reviewer_name}</p>
                </div>
              </div>
            )}

            <div className="space-y-1">
              <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                Review Due
              </p>
              <div className="flex items-center gap-1.5">
                <Calendar className="h-3.5 w-3.5 text-muted-foreground" />
                <p
                  className={cn(
                    'text-sm',
                    isOverdue && 'text-red-600 font-medium',
                  )}
                >
                  {formatDate(policy.review_due)}
                  {isOverdue && ' (Overdue)'}
                </p>
              </div>
            </div>

            {policy.last_reviewed_at && (
              <div className="space-y-1">
                <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  Last Reviewed
                </p>
                <div className="flex items-center gap-1.5">
                  <Clock className="h-3.5 w-3.5 text-muted-foreground" />
                  <p className="text-sm">{formatDate(policy.last_reviewed_at)}</p>
                </div>
              </div>
            )}

            {policy.approved_by_name && (
              <div className="space-y-1">
                <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  Approved By
                </p>
                <div className="flex items-center gap-1.5">
                  <CheckCircle className="h-3.5 w-3.5 text-green-600" />
                  <p className="text-sm">{policy.approved_by_name}</p>
                </div>
              </div>
            )}

            {policy.approved_at && (
              <div className="space-y-1">
                <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                  Approved At
                </p>
                <p className="text-sm">{formatDate(policy.approved_at)}</p>
              </div>
            )}

            <div className="space-y-1">
              <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                Exceptions
              </p>
              <p className="text-sm">
                {policy.exceptions_count > 0 ? (
                  <span className="font-medium text-orange-600">
                    {policy.exceptions_count} active
                  </span>
                ) : (
                  <span className="text-muted-foreground">None</span>
                )}
              </p>
            </div>

            <div className="space-y-1">
              <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                Last Updated
              </p>
              <p className="text-sm">{formatDate(policy.updated_at)}</p>
            </div>
          </div>

          {/* Tags */}
          {policy.tags && policy.tags.length > 0 && (
            <>
              <Separator />
              <div className="space-y-2">
                <div className="flex items-center gap-1.5">
                  <Tag className="h-3.5 w-3.5 text-muted-foreground" />
                  <p className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    Tags
                  </p>
                </div>
                <div className="flex flex-wrap gap-1.5">
                  {policy.tags.map((tag) => (
                    <Badge key={tag} variant="secondary">
                      {tag}
                    </Badge>
                  ))}
                </div>
              </div>
            </>
          )}

          <Separator />

          {/* Policy Content */}
          <div className="space-y-3">
            <h3 className="text-sm font-semibold text-foreground">Policy Content</h3>
            <div className="rounded-lg border border-border bg-muted/30 p-4">
              <div className="prose prose-sm max-w-none whitespace-pre-wrap text-sm leading-relaxed text-foreground">
                {policy.content}
              </div>
            </div>
          </div>
        </div>
      </DetailPanel>

      {confirmAction && (
        <ConfirmDialog
          open={!!confirmAction}
          onOpenChange={(o) => !o && setConfirmAction(null)}
          title={confirmAction.title}
          description={confirmAction.description}
          confirmLabel={confirmAction.title}
          onConfirm={handleStatusChange}
          loading={statusMutation.isPending}
        />
      )}
    </>
  );
}
