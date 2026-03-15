'use client';

import { useQuery } from '@tanstack/react-query';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Badge } from '@/components/ui/badge';
import { enterpriseApi } from '@/lib/enterprise';
import { RelativeTime } from '@/components/shared/relative-time';

interface SnapshotPreviewDialogProps {
  reportId: string | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function SnapshotPreviewDialog({ reportId, open, onOpenChange }: SnapshotPreviewDialogProps) {
  const { data: snapshot, isLoading, error } = useQuery({
    queryKey: ['visus-snapshot-preview', reportId],
    queryFn: () => enterpriseApi.visus.getLatestReportSnapshot(reportId!),
    enabled: open && !!reportId,
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Latest Snapshot</DialogTitle>
          <DialogDescription>Most recent report generation output.</DialogDescription>
        </DialogHeader>
        {isLoading && <p className="py-4 text-sm text-muted-foreground">Loading snapshot...</p>}
        {error && <p className="py-4 text-sm text-destructive">Failed to load snapshot.</p>}
        {snapshot && (
          <div className="space-y-4">
            <div className="grid grid-cols-1 gap-3 text-sm sm:grid-cols-2">
              <div>
                <p className="text-muted-foreground">Period</p>
                <p className="font-medium">{snapshot.period_start} — {snapshot.period_end}</p>
              </div>
              <div>
                <p className="text-muted-foreground">Generated</p>
                <p className="font-medium"><RelativeTime date={snapshot.generated_at} /></p>
              </div>
              {snapshot.generation_time_ms != null && (
                <div>
                  <p className="text-muted-foreground">Generation Time</p>
                  <p className="font-medium">{snapshot.generation_time_ms}ms</p>
                </div>
              )}
              <div>
                <p className="text-muted-foreground">Format</p>
                <p className="font-medium uppercase">{snapshot.file_format}</p>
              </div>
            </div>
            {snapshot.sections_included.length > 0 && (
              <div>
                <p className="mb-1 text-sm text-muted-foreground">Sections</p>
                <div className="flex flex-wrap gap-1">
                  {snapshot.sections_included.map((section) => (
                    <Badge key={section} variant="outline" className="capitalize">
                      {section.replace(/_/g, ' ')}
                    </Badge>
                  ))}
                </div>
              </div>
            )}
            {snapshot.narrative && (
              <div>
                <p className="mb-1 text-sm text-muted-foreground">Narrative</p>
                <p className="rounded-md border bg-muted/50 p-3 text-sm">{snapshot.narrative}</p>
              </div>
            )}
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
