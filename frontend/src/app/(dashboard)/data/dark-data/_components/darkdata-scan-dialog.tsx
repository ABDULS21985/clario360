'use client';

import { useEffect, useState } from 'react';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Progress } from '@/components/ui/progress';
import { Spinner } from '@/components/ui/spinner';
import { dataSuiteApi, type DarkDataScan } from '@/lib/data-suite';

interface DarkDataScanDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onComplete?: () => void;
}

export function DarkDataScanDialog({
  open,
  onOpenChange,
  onComplete,
}: DarkDataScanDialogProps) {
  const [scan, setScan] = useState<DarkDataScan | null>(null);

  useEffect(() => {
    if (!open) {
      setScan(null);
      return;
    }

    let cancelled = false;
    let timer: number | null = null;

    const start = async () => {
      const initial = await dataSuiteApi.scanDarkData();
      if (cancelled) {
        return;
      }
      setScan(initial);

      const poll = async () => {
        const current = await dataSuiteApi.getDarkDataScan(initial.id);
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

  const progress = scan ? (scan.status === 'running' ? Math.min(90, scan.assets_discovered * 10) : 100) : 10;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Dark Data Scan</DialogTitle>
        </DialogHeader>
        {!scan ? (
          <div className="flex items-center gap-3 rounded-lg border bg-muted/20 p-4">
            <Spinner />
            <div className="text-sm">Starting dark data scan…</div>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="text-sm capitalize">Status: {scan.status}</div>
            <Progress value={progress} />
            <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
              <Metric label="Sources Scanned" value={scan.sources_scanned.toLocaleString()} />
              <Metric label="Assets Found" value={scan.assets_discovered.toLocaleString()} />
              <Metric label="PII Assets" value={scan.pii_assets_found.toLocaleString()} />
              <Metric label="High Risk" value={scan.high_risk_found.toLocaleString()} />
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
