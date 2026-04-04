'use client';

import { useEffect, useState, useCallback } from 'react';
import { CheckCircle, XCircle, Download, Loader2 } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { ExportJob } from '@/types/cyber';

type ExportState = 'preparing' | 'downloading' | 'complete' | 'failed';

interface ExportProgressDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  jobId: string | null;
}

function downloadUrl(url: string, filename: string) {
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
}

export function ExportProgressDialog({ open, onOpenChange, jobId }: ExportProgressDialogProps) {
  const [state, setState] = useState<ExportState>('preparing');
  const [progress, setProgress] = useState(0);
  const [downloadHref, setDownloadHref] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const poll = useCallback(async (id: string) => {
    try {
      const res = await apiGet<{ data: ExportJob }>(`${API_ENDPOINTS.JOBS}/${id}`);
      const job = res.data;
      if (job.status === 'completed' && job.download_url) {
        setState('complete');
        setDownloadHref(job.download_url);
        setProgress(100);
        // Auto-trigger download
        downloadUrl(job.download_url, 'export.pdf');
        // Auto-close after 3 seconds
        setTimeout(() => onOpenChange(false), 3000);
      } else if (job.status === 'failed') {
        setState('failed');
        setError(job.error ?? 'Export failed');
      } else {
        setState('downloading');
        setProgress(job.progress ?? 50);
      }
    } catch {
      setState('failed');
      setError('Failed to check export status');
    }
  }, [onOpenChange]);

  useEffect(() => {
    if (!open || !jobId) {
      setState('preparing');
      setProgress(0);
      setDownloadHref(null);
      setError(null);
      return;
    }

    const interval = setInterval(() => {
      void poll(jobId);
    }, 3000);

    // Initial poll
    void poll(jobId);

    return () => clearInterval(interval);
  }, [open, jobId, poll]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>Exporting Report</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 py-2">
          {state === 'preparing' && (
            <>
              <div className="flex items-center gap-3 text-muted-foreground">
                <Loader2 className="h-5 w-5 animate-spin" />
                <span className="text-sm">Preparing export…</span>
              </div>
              <Progress value={undefined} className="h-1.5" />
            </>
          )}
          {state === 'downloading' && (
            <>
              <div className="flex items-center gap-3 text-muted-foreground">
                <Loader2 className="h-5 w-5 animate-spin" />
                <span className="text-sm">Generating PDF…</span>
              </div>
              <Progress value={progress} className="h-1.5" />
            </>
          )}
          {state === 'complete' && (
            <div className="flex items-center gap-3 text-green-600">
              <CheckCircle className="h-5 w-5" />
              <span className="text-sm font-medium">Export ready</span>
            </div>
          )}
          {state === 'failed' && (
            <div className="space-y-2">
              <div className="flex items-center gap-3 text-destructive">
                <XCircle className="h-5 w-5" />
                <span className="text-sm font-medium">Export failed</span>
              </div>
              {error && <p className="text-xs text-muted-foreground">{error}</p>}
            </div>
          )}

          <div className="flex justify-end gap-2">
            {state === 'complete' && downloadHref && (
              <Button size="sm" onClick={() => downloadUrl(downloadHref, 'export.pdf')}>
                <Download className="mr-1.5 h-3.5 w-3.5" />
                Download
              </Button>
            )}
            {state === 'failed' && (
              <Button size="sm" variant="outline" onClick={() => onOpenChange(false)}>
                Close
              </Button>
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
