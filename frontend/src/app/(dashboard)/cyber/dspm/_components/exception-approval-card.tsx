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
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  CheckCircle2,
  XCircle,
  Clock,
  AlertTriangle,
  Shield,
  Calendar,
  RefreshCw,
} from 'lucide-react';
import type { DSPMRiskException } from '@/types/cyber';

interface ExceptionApprovalCardProps {
  exception: DSPMRiskException;
  onApprove: (id: string) => void;
  onReject: (id: string, reason: string) => void;
}

const APPROVAL_BADGE: Record<string, { label: string; class: string; icon: typeof CheckCircle2 }> = {
  pending: { label: 'Pending', class: 'bg-amber-100 text-amber-800', icon: Clock },
  approved: { label: 'Approved', class: 'bg-green-100 text-green-700', icon: CheckCircle2 },
  rejected: { label: 'Rejected', class: 'bg-red-100 text-red-700', icon: XCircle },
  expired: { label: 'Expired', class: 'bg-gray-100 text-gray-700', icon: Clock },
};

function getRiskColor(score: number): string {
  if (score >= 80) return 'text-red-600';
  if (score >= 60) return 'text-orange-600';
  if (score >= 40) return 'text-amber-600';
  return 'text-green-600';
}

function getRiskBg(score: number): string {
  if (score >= 80) return 'bg-red-50 dark:bg-red-950/20';
  if (score >= 60) return 'bg-orange-50 dark:bg-orange-950/20';
  if (score >= 40) return 'bg-amber-50 dark:bg-amber-950/20';
  return 'bg-green-50 dark:bg-green-950/20';
}

function formatDate(ts?: string): string {
  if (!ts) return '---';
  return new Date(ts).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  });
}

function formatExceptionType(type: string): string {
  return type.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
}

export function ExceptionApprovalCard({ exception, onApprove, onReject }: ExceptionApprovalCardProps) {
  const [showRejectInput, setShowRejectInput] = useState(false);
  const [rejectReason, setRejectReason] = useState('');

  const badge = APPROVAL_BADGE[exception.approval_status] ?? APPROVAL_BADGE.pending;
  const BadgeIcon = badge.icon;

  const handleReject = () => {
    if (!rejectReason.trim()) return;
    onReject(exception.id, rejectReason.trim());
    setShowRejectInput(false);
    setRejectReason('');
  };

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between gap-3">
          <CardTitle className="text-base">{formatExceptionType(exception.exception_type)}</CardTitle>
          <span className={`inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-medium ${badge.class}`}>
            <BadgeIcon className="h-3 w-3" />
            {badge.label}
          </span>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Risk Score */}
        <div className={`flex items-center gap-3 rounded-lg p-3 ${getRiskBg(exception.risk_score)}`}>
          <Shield className={`h-5 w-5 ${getRiskColor(exception.risk_score)}`} />
          <div>
            <p className="text-xs text-muted-foreground">Risk Score</p>
            <p className={`text-lg font-bold tabular-nums ${getRiskColor(exception.risk_score)}`}>
              {exception.risk_score}/100
            </p>
          </div>
          <Badge variant="outline" className="ml-auto capitalize">{exception.risk_level}</Badge>
        </div>

        {/* Details */}
        <div className="space-y-3 text-sm">
          <div>
            <p className="font-medium text-muted-foreground">Justification</p>
            <p className="mt-0.5">{exception.justification}</p>
          </div>

          {exception.business_reason && (
            <div>
              <p className="font-medium text-muted-foreground">Business Reason</p>
              <p className="mt-0.5">{exception.business_reason}</p>
            </div>
          )}

          {exception.compensating_controls && (
            <div>
              <p className="font-medium text-muted-foreground">Compensating Controls</p>
              <p className="mt-0.5">{exception.compensating_controls}</p>
            </div>
          )}

          <div>
            <p className="font-medium text-muted-foreground">Requested By</p>
            <p className="mt-0.5">{exception.requested_by}</p>
          </div>
        </div>

        {/* Dates */}
        <div className="grid grid-cols-2 gap-3 text-xs">
          <div className="flex items-center gap-1.5">
            <Calendar className="h-3.5 w-3.5 text-muted-foreground" />
            <span className="text-muted-foreground">Expires:</span>
            <span className="font-medium">{formatDate(exception.expires_at)}</span>
          </div>
          {exception.next_review_at && (
            <div className="flex items-center gap-1.5">
              <RefreshCw className="h-3.5 w-3.5 text-muted-foreground" />
              <span className="text-muted-foreground">Next review:</span>
              <span className="font-medium">{formatDate(exception.next_review_at)}</span>
            </div>
          )}
          <div className="flex items-center gap-1.5">
            <span className="text-muted-foreground">Reviews:</span>
            <span className="font-medium">{exception.review_count}</span>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="text-muted-foreground">Interval:</span>
            <span className="font-medium">{exception.review_interval_days} days</span>
          </div>
        </div>

        {/* Approved info */}
        {exception.approval_status === 'approved' && exception.approved_by && (
          <div className="rounded-lg border border-green-200 bg-green-50 p-3 text-sm dark:border-green-900 dark:bg-green-950/20">
            <div className="flex items-center gap-2">
              <CheckCircle2 className="h-4 w-4 text-green-600" />
              <span className="font-medium text-green-700 dark:text-green-400">Approved</span>
            </div>
            <p className="mt-1 text-xs text-muted-foreground">
              By {exception.approved_by} on {formatDate(exception.approved_at)}
            </p>
          </div>
        )}

        {/* Rejected info */}
        {exception.approval_status === 'rejected' && (
          <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm dark:border-red-900 dark:bg-red-950/20">
            <div className="flex items-center gap-2">
              <XCircle className="h-4 w-4 text-red-600" />
              <span className="font-medium text-red-700 dark:text-red-400">Rejected</span>
            </div>
            {exception.rejection_reason && (
              <p className="mt-1 text-xs text-muted-foreground">{exception.rejection_reason}</p>
            )}
          </div>
        )}

        {/* Expired info */}
        {exception.approval_status === 'expired' && (
          <div className="rounded-lg border bg-muted/50 p-3 text-sm">
            <div className="flex items-center gap-2">
              <AlertTriangle className="h-4 w-4 text-muted-foreground" />
              <span className="font-medium text-muted-foreground">Exception Expired</span>
            </div>
          </div>
        )}

        {/* Approval actions */}
        {exception.approval_status === 'pending' && (
          <div className="space-y-3 border-t pt-3">
            {showRejectInput ? (
              <div className="space-y-2">
                <Label htmlFor="reject-reason" className="text-xs">Rejection Reason (required)</Label>
                <Input
                  id="reject-reason"
                  placeholder="Provide a reason for rejection..."
                  value={rejectReason}
                  onChange={(e) => setRejectReason(e.target.value)}
                />
                <div className="flex items-center gap-2">
                  <Button
                    type="button"
                    size="sm"
                    variant="destructive"
                    disabled={!rejectReason.trim()}
                    onClick={handleReject}
                  >
                    Confirm Reject
                  </Button>
                  <Button
                    type="button"
                    size="sm"
                    variant="ghost"
                    onClick={() => { setShowRejectInput(false); setRejectReason(''); }}
                  >
                    Cancel
                  </Button>
                </div>
              </div>
            ) : (
              <div className="flex items-center gap-2">
                <Button
                  type="button"
                  size="sm"
                  className="bg-green-600 hover:bg-green-700 text-white"
                  onClick={() => onApprove(exception.id)}
                >
                  <CheckCircle2 className="mr-1.5 h-3.5 w-3.5" />
                  Approve
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant="destructive"
                  onClick={() => setShowRejectInput(true)}
                >
                  <XCircle className="mr-1.5 h-3.5 w-3.5" />
                  Reject
                </Button>
              </div>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
