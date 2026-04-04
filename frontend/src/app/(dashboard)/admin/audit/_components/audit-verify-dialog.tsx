'use client';

import { useState } from 'react';
import { format, subDays } from 'date-fns';
import { CheckCircle, XCircle, ShieldCheck } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import { Progress } from '@/components/ui/progress';
import { apiPost } from '@/lib/api';
import { isApiError } from '@/types/api';
import { formatNumber } from '@/lib/format';
import type { AuditVerificationResult } from '@/types/audit';

interface AuditVerifyDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function AuditVerifyDialog({ open, onOpenChange }: AuditVerifyDialogProps) {
  const today = format(new Date(), 'yyyy-MM-dd');
  const thirtyDaysAgo = format(subDays(new Date(), 30), 'yyyy-MM-dd');

  const [dateFrom, setDateFrom] = useState(thirtyDaysAgo);
  const [dateTo, setDateTo] = useState(today);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<AuditVerificationResult | null>(null);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const handleVerify = async () => {
    setLoading(true);
    setResult(null);
    setErrorMsg(null);
    try {
      const res = await apiPost<AuditVerificationResult>('/api/v1/audit/verify', {
        date_from: new Date(dateFrom).toISOString(),
        date_to: new Date(dateTo + 'T23:59:59').toISOString(),
      });
      setResult(res);
    } catch (err) {
      setErrorMsg(isApiError(err) ? err.message : 'Verification failed.');
    } finally {
      setLoading(false);
    }
  };

  const handleClose = (open: boolean) => {
    if (!open) {
      setResult(null);
      setErrorMsg(null);
      setDateFrom(thirtyDaysAgo);
      setDateTo(today);
    }
    onOpenChange(open);
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary/10">
              <ShieldCheck className="h-5 w-5 text-primary" />
            </div>
            <div>
              <DialogTitle>Verify Audit Integrity</DialogTitle>
              <DialogDescription>
                Verify the hash chain integrity of audit records.
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className="space-y-4">
          {!loading && !result && !errorMsg && (
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="date-from">From</Label>
                <Input
                  id="date-from"
                  type="date"
                  value={dateFrom}
                  onChange={(e) => setDateFrom(e.target.value)}
                  max={dateTo}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="date-to">To</Label>
                <Input
                  id="date-to"
                  type="date"
                  value={dateTo}
                  onChange={(e) => setDateTo(e.target.value)}
                  min={dateFrom}
                  max={today}
                />
              </div>
            </div>
          )}

          {loading && (
            <div className="space-y-3 py-4">
              <p className="text-sm text-center text-muted-foreground">
                Verifying hash chain integrity...
              </p>
              <Progress value={undefined} className="h-2 animate-pulse" />
            </div>
          )}

          {errorMsg && (
            <div className="rounded-lg border border-destructive/50 p-4">
              <div className="flex items-center gap-2 text-destructive">
                <XCircle className="h-5 w-5" />
                <span className="font-medium">Verification Error</span>
              </div>
              <p className="mt-1 text-sm text-muted-foreground">{errorMsg}</p>
            </div>
          )}

          {result && (
            <div className="rounded-lg border p-4">
              {result.verified ? (
                <div className="space-y-2">
                  <div className="flex items-center gap-2 text-green-600 dark:text-green-400">
                    <CheckCircle className="h-5 w-5" />
                    <span className="font-medium">Integrity Verified</span>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    {formatNumber(result.verified_records)} of {formatNumber(result.total_records)} records verified.
                  </p>
                </div>
              ) : (
                <div className="space-y-2">
                  <div className="flex items-center gap-2 text-destructive">
                    <XCircle className="h-5 w-5" />
                    <span className="font-medium">Integrity Violation Detected</span>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    {result.verified_records > 0
                      ? `${formatNumber(result.verified_records)} records verified before the break was detected.`
                      : 'Chain integrity check failed.'}
                  </p>
                  {result.broken_chain_at && (
                    <p className="text-xs font-mono text-destructive">
                      Broken at entry: {result.broken_chain_at}
                    </p>
                  )}
                </div>
              )}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => handleClose(false)}>
            Close
          </Button>
          {!result && !errorMsg && (
            <Button onClick={handleVerify} disabled={loading || !dateFrom || !dateTo}>
              {loading ? 'Verifying...' : 'Start Verification'}
            </Button>
          )}
          {(result || errorMsg) && (
            <Button variant="outline" onClick={() => { setResult(null); setErrorMsg(null); }}>
              Verify Again
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
