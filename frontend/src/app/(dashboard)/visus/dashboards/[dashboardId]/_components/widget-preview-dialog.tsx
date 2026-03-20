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
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { enterpriseApi } from '@/lib/enterprise';
import type { VisusWidget } from '@/types/suites';
import { formatJsonInput } from '../../../_components/form-utils';

interface WidgetPreviewDialogProps {
  dashboardId: string;
  widget: VisusWidget | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function WidgetPreviewDialog({ dashboardId, widget, open, onOpenChange }: WidgetPreviewDialogProps) {
  const previewQuery = useQuery({
    queryKey: ['visus-widget-preview', dashboardId, widget?.id],
    queryFn: () => enterpriseApi.visus.getWidgetData(dashboardId, widget!.id),
    enabled: open && Boolean(widget),
  });

  const data = previewQuery.data;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] max-w-3xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{widget?.title ?? 'Widget preview'}</DialogTitle>
          <DialogDescription>Live data payload returned by the widget endpoint.</DialogDescription>
        </DialogHeader>

        {previewQuery.isLoading ? <p className="text-sm text-muted-foreground">Loading widget data...</p> : null}
        {previewQuery.isError ? <p className="text-sm text-destructive">Unable to load widget data.</p> : null}

        {data ? (
          <div className="space-y-4">
            {'value' in data && typeof data.value === 'number' ? (
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                <div className="rounded-xl border p-4">
                  <p className="text-xs uppercase tracking-wide text-muted-foreground">Value</p>
                  <p className="mt-1 text-2xl font-semibold">{data.value}</p>
                </div>
                {'status' in data && typeof data.status === 'string' ? (
                  <div className="rounded-xl border p-4">
                    <p className="text-xs uppercase tracking-wide text-muted-foreground">Status</p>
                    <div className="mt-2">
                      <Badge variant="outline">{data.status}</Badge>
                    </div>
                  </div>
                ) : null}
                {'delta_percent' in data && typeof data.delta_percent === 'number' ? (
                  <div className="rounded-xl border p-4">
                    <p className="text-xs uppercase tracking-wide text-muted-foreground">Delta %</p>
                    <p className="mt-1 text-2xl font-semibold">{data.delta_percent.toFixed(2)}%</p>
                  </div>
                ) : null}
              </div>
            ) : null}

            {'alerts' in data && Array.isArray(data.alerts) ? (
              <div className="space-y-2">
                <p className="text-sm font-medium">Alert Feed</p>
                <div className="space-y-2">
                  {data.alerts.map((alert) => (
                    <div key={alert.id} className="rounded-lg border p-3">
                      <div className="flex items-center justify-between gap-3">
                        <p className="font-medium">{alert.title}</p>
                        <Badge variant="outline">{alert.severity}</Badge>
                      </div>
                      <p className="mt-1 text-sm text-muted-foreground">{alert.description}</p>
                    </div>
                  ))}
                </div>
              </div>
            ) : null}

            {'columns' in data && Array.isArray(data.columns) && 'rows' in data && Array.isArray(data.rows) ? (
              <div className="space-y-2">
                <p className="text-sm font-medium">Table Preview</p>
                <div className="overflow-x-auto rounded-lg border">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        {data.columns.map((column) => (
                          <TableHead key={column.key}>{column.label}</TableHead>
                        ))}
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {data.rows.slice(0, 5).map((row, index) => (
                        <TableRow key={index}>
                          {data.columns.map((column) => (
                            <TableCell key={column.key}>{String(row[column.key] ?? '—')}</TableCell>
                          ))}
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              </div>
            ) : null}

            {'items' in data && Array.isArray(data.items) ? (
              <div className="space-y-2">
                <p className="text-sm font-medium">Status Items</p>
                <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                  {data.items.map((item, index) => (
                    <div key={`${item.label}-${index}`} className="rounded-lg border p-3">
                      <div className="flex items-center justify-between gap-2">
                        <p className="font-medium">{item.label}</p>
                        <Badge variant="outline">{item.status}</Badge>
                      </div>
                      <p className="mt-2 text-lg font-semibold">
                        {item.value}
                        {item.unit ? ` ${item.unit}` : ''}
                      </p>
                    </div>
                  ))}
                </div>
              </div>
            ) : null}

            {'content' in data && typeof data.content === 'string' ? (
              <div className="rounded-lg border bg-muted/30 p-4 text-sm leading-7">{data.content}</div>
            ) : null}

            <div className="space-y-2">
              <p className="text-sm font-medium">Raw Payload</p>
              <pre className="overflow-x-auto rounded-lg border bg-muted/40 p-4 text-xs">
                {formatJsonInput(data)}
              </pre>
            </div>
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
