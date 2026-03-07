'use client';

import { useEffect, useState } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Progress } from '@/components/ui/progress';
import { dataSuiteApi, type ContradictionScan } from '@/lib/data-suite';
import { Spinner } from '@/components/ui/spinner';

interface ContradictionScanDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onComplete?: () => void;
}

export function ContradictionScanDialog({
  open,
  onOpenChange,
  onComplete,
}: ContradictionScanDialogProps) {
  const [scan, setScan] = useState<ContradictionScan | null>(null);

  useEffect(() => {
    if (!open) {
      setScan(null);
      return;
    }

    let cancelled = false;
    let timer: number | null = null;

    const start = async () => {
      const initial = await dataSuiteApi.scanContradictions();
      if (cancelled) {
        return;
      }
      setScan(initial);

      const poll = async () => {
        if (!initial.id || cancelled) {
          return;
        }
        const current = await dataSuiteApi.getContradictionScan(initial.id);
        if (cancelled) {
          return;
        }
        setScan(current);
        if (current.status === 'completed' || current.status === 'failed') {
          onComplete?.();
          return;
        }
        timer = window.setTimeout(() => void poll(), 3000);
      };

      timer = window.setTimeout(() => void poll(), 3000);
    };

    void start();

    return () => {
      cancelled = true;
      if (timer) {
        window.clearTimeout(timer);
      }
    };
  }, [onComplete, open]);

  const progress = scan ? (scan.status === 'running' ? Math.min(90, scan.models_scanned * 10) : 100) : 10;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Contradiction Scan</DialogTitle>
        </DialogHeader>
        {!scan ? (
          <div className="flex items-center gap-3 rounded-lg border bg-muted/20 p-4">
            <Spinner />
            <div className="text-sm">Starting contradiction scan…</div>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="text-sm capitalize">Status: {scan.status}</div>
            <Progress value={progress} />
            <div className="grid gap-3 md:grid-cols-2">
              <Metric label="Models Scanned" value={scan.models_scanned.toLocaleString()} />
              <Metric label="Pairs Compared" value={scan.model_pairs_compared.toLocaleString()} />
              <Metric label="Found" value={scan.contradictions_found.toLocaleString()} />
              <Metric label="Triggered By" value={scan.triggered_by} />
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

function Metric({
  label,
  value,
}: {
  label: string;
  value: string;
}) {
  return (
    <div className="rounded-lg border bg-muted/20 p-3">
      <div className="text-xs uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className="mt-1 text-sm font-medium">{value}</div>
    </div>
  );
}
