'use client';

import { useState } from 'react';
import { Download, FileText, FileJson, FileBarChart2, ChevronDown, AlertCircle } from 'lucide-react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { apiGet, apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { toast } from 'sonner';
import { ExportProgressDialog } from './export-progress-dialog';

export interface ExportMenuProps {
  entityType: string;
  baseUrl: string;
  currentFilters: Record<string, string | string[]>;
  totalCount: number;
  pdfReportUrl?: string;
  enabledFormats?: ('csv' | 'json' | 'pdf')[];
  selectedCount?: number;
  getSelectedIds?: () => string[];
}

const EXPORT_LIMIT = 50000;
const EXPORT_WARN_AT = 10000;

function buildParams(
  filters: Record<string, string | string[]>,
  perPage: number,
): Record<string, unknown> {
  const params: Record<string, unknown> = { ...filters, per_page: perPage };
  return params;
}

function jsonToCsv(data: unknown[]): string {
  if (data.length === 0) return '';
  const headers = Object.keys(data[0] as Record<string, unknown>);
  const rows = data.map((row) =>
    headers
      .map((h) => {
        const val = (row as Record<string, unknown>)[h];
        const str = val === null || val === undefined ? '' : String(val);
        return str.includes(',') || str.includes('"') || str.includes('\n')
          ? `"${str.replace(/"/g, '""')}"`
          : str;
      })
      .join(','),
  );
  return [headers.join(','), ...rows].join('\n');
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

function dateSuffix(): string {
  return new Date().toISOString().slice(0, 10);
}

export function ExportMenu({
  entityType,
  baseUrl,
  currentFilters,
  totalCount,
  pdfReportUrl,
  enabledFormats = ['csv', 'json', 'pdf'],
  selectedCount = 0,
}: ExportMenuProps) {
  const [warnOpen, setWarnOpen] = useState(false);
  const [blockedOpen, setBlockedOpen] = useState(false);
  const [pendingFormat, setPendingFormat] = useState<'csv' | 'json' | null>(null);
  const [exportJobId, setExportJobId] = useState<string | null>(null);
  const [exportProgressOpen, setExportProgressOpen] = useState(false);
  const [exporting, setExporting] = useState(false);

  async function doExport(format: 'csv' | 'json') {
    setExporting(true);
    try {
      const params = buildParams(currentFilters, EXPORT_LIMIT);
      if (format === 'csv') {
        // Try Accept header first
        try {
          const res = await fetch(`${baseUrl}?${new URLSearchParams(Object.entries(params).map(([k, v]) => [k, String(v)]))}`, {
            headers: { Accept: 'text/csv' },
          });
          if (res.ok && res.headers.get('content-type')?.includes('text/csv')) {
            const text = await res.text();
            downloadBlob(
              new Blob([text], { type: 'text/csv;charset=utf-8;' }),
              `${entityType}-export-${dateSuffix()}.csv`,
            );
            toast.success('CSV export ready');
            return;
          }
        } catch {
          // fallthrough to JSON-based CSV
        }
        // Fallback: fetch JSON and convert
        const data = await apiGet<{ data: unknown[] }>(baseUrl, params);
        const rows = Array.isArray(data) ? data : (data.data ?? []);
        const csv = jsonToCsv(rows as unknown[]);
        downloadBlob(
          new Blob([csv], { type: 'text/csv;charset=utf-8;' }),
          `${entityType}-export-${dateSuffix()}.csv`,
        );
        toast.success('CSV export ready');
      } else {
        const data = await apiGet<unknown>(baseUrl, params);
        const json = JSON.stringify(data, null, 2);
        downloadBlob(
          new Blob([json], { type: 'application/json' }),
          `${entityType}-export-${dateSuffix()}.json`,
        );
        toast.success('JSON export ready');
      }
    } catch {
      toast.error('Export failed');
    } finally {
      setExporting(false);
    }
  }

  function handleExportClick(format: 'csv' | 'json') {
    const count = totalCount ?? 0;
    if (count > EXPORT_LIMIT) {
      setBlockedOpen(true);
      return;
    }
    if (count > EXPORT_WARN_AT) {
      setPendingFormat(format);
      setWarnOpen(true);
      return;
    }
    void doExport(format);
  }

  async function handlePdfExport() {
    if (!pdfReportUrl) return;
    try {
      const res = await apiPost<{ data: { job_id: string } }>(pdfReportUrl);
      setExportJobId(res.data.job_id);
      setExportProgressOpen(true);
    } catch {
      toast.error('Failed to start PDF export');
    }
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="sm" disabled={exporting}>
            <Download className="mr-1.5 h-3.5 w-3.5" />
            Export
            <ChevronDown className="ml-1 h-3.5 w-3.5" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-64">
          {enabledFormats.includes('csv') && (
            <DropdownMenuItem onClick={() => handleExportClick('csv')}>
              <FileText className="mr-2 h-4 w-4 text-green-600" />
              Export as CSV ({(totalCount ?? 0).toLocaleString()} records)
            </DropdownMenuItem>
          )}
          {enabledFormats.includes('json') && (
            <DropdownMenuItem onClick={() => handleExportClick('json')}>
              <FileJson className="mr-2 h-4 w-4 text-blue-600" />
              Export as JSON ({(totalCount ?? 0).toLocaleString()} records)
            </DropdownMenuItem>
          )}
          {enabledFormats.includes('pdf') && pdfReportUrl && (
            <DropdownMenuItem onClick={() => void handlePdfExport()}>
              <FileBarChart2 className="mr-2 h-4 w-4 text-red-600" />
              Generate PDF Report
            </DropdownMenuItem>
          )}
          {selectedCount > 0 && (
            <>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => handleExportClick('csv')}>
                <Download className="mr-2 h-4 w-4" />
                Export Selected ({(selectedCount ?? 0).toLocaleString()} records)
              </DropdownMenuItem>
            </>
          )}
        </DropdownMenuContent>
      </DropdownMenu>

      {/* Large export warning dialog */}
      <Dialog open={warnOpen} onOpenChange={setWarnOpen}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <AlertCircle className="h-5 w-5 text-yellow-500" />
              Large Export
            </DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            You are about to export <strong>{(totalCount ?? 0).toLocaleString()}</strong> records. This may
            take a moment. Consider applying filters to reduce the dataset.
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setWarnOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => {
                setWarnOpen(false);
                if (pendingFormat) void doExport(pendingFormat);
              }}
            >
              Export Anyway
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Blocked export dialog */}
      <Dialog open={blockedOpen} onOpenChange={setBlockedOpen}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <AlertCircle className="h-5 w-5 text-destructive" />
              Export Limit Exceeded
            </DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            Export limit is <strong>50,000 records</strong>. Please apply filters to reduce the
            dataset. Current filter returns <strong>{(totalCount ?? 0).toLocaleString()}</strong> records.
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setBlockedOpen(false)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* PDF export progress */}
      <ExportProgressDialog
        open={exportProgressOpen}
        onOpenChange={setExportProgressOpen}
        jobId={exportJobId}
      />
    </>
  );
}
