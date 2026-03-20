'use client';

import { useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import { Upload, AlertCircle } from 'lucide-react';
import type { CyberAsset } from '@/types/cyber';

interface BulkImportResult {
  created: number;
  updated: number;
  failed: number;
  errors?: string[];
}

interface BulkImportDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: (result: BulkImportResult) => void;
}

const EXAMPLE = JSON.stringify([
  { name: 'web-prod-01', type: 'server', criticality: 'high', ip_address: '10.0.1.10', hostname: 'web-prod-01.example.com', os: 'Ubuntu 22.04', owner: 'Infra Team' },
  { name: 'db-prod-01', type: 'database', criticality: 'critical', ip_address: '10.0.2.10', owner: 'DBA Team' },
], null, 2);

export function BulkImportDialog({ open, onOpenChange, onSuccess }: BulkImportDialogProps) {
  const [raw, setRaw] = useState('');
  const [parseError, setParseError] = useState<string | null>(null);
  const [preview, setPreview] = useState<Partial<CyberAsset>[] | null>(null);

  const { mutate, isPending } = useApiMutation<BulkImportResult, { assets: Partial<CyberAsset>[] }>(
    'post',
    API_ENDPOINTS.CYBER_ASSETS_BULK,
    {
      successMessage: 'Bulk import complete',
      invalidateKeys: ['cyber-assets', 'cyber-assets-stats'],
      onSuccess: (result) => {
        setRaw('');
        setPreview(null);
        onOpenChange(false);
        onSuccess?.(result);
      },
    },
  );

  const handleParse = () => {
    setParseError(null);
    try {
      const parsed = JSON.parse(raw) as unknown;
      if (!Array.isArray(parsed)) {
        setParseError('Input must be a JSON array of assets');
        return;
      }
      setPreview(parsed as Partial<CyberAsset>[]);
    } catch (e) {
      setParseError(`Invalid JSON: ${(e as Error).message}`);
    }
  };

  const handleImport = () => {
    if (!preview) return;
    mutate({ assets: preview });
  };

  const handleClose = () => {
    setRaw('');
    setPreview(null);
    setParseError(null);
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>Bulk Import Assets</DialogTitle>
          <DialogDescription>
            Paste a JSON array of assets to import. Required fields: <code className="text-xs">name</code>, <code className="text-xs">type</code>, <code className="text-xs">criticality</code>.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {!preview ? (
            <>
              <div className="space-y-2">
                <Label htmlFor="bulk-json">JSON Input</Label>
                <Textarea
                  id="bulk-json"
                  value={raw}
                  onChange={(e) => setRaw(e.target.value)}
                  placeholder={EXAMPLE}
                  className="h-48 font-mono text-xs"
                />
              </div>

              {parseError && (
                <div className="flex items-center gap-2 rounded-md bg-destructive/10 p-3 text-sm text-destructive">
                  <AlertCircle className="h-4 w-4 shrink-0" />
                  {parseError}
                </div>
              )}

              <Button type="button" variant="outline" onClick={handleParse} disabled={!raw.trim()}>
                Validate & Preview
              </Button>
            </>
          ) : (
            <>
              <div className="rounded-md border">
                <div className="flex items-center justify-between border-b px-3 py-2">
                  <p className="text-sm font-medium">Preview ({preview.length} assets)</p>
                  <Button type="button" variant="ghost" size="sm" onClick={() => setPreview(null)}>
                    Edit
                  </Button>
                </div>
                <div className="max-h-48 overflow-y-auto">
                  <table className="w-full text-xs">
                    <thead className="bg-muted/50">
                      <tr>
                        <th className="px-3 py-2 text-left">Name</th>
                        <th className="px-3 py-2 text-left">Type</th>
                        <th className="px-3 py-2 text-left">Criticality</th>
                        <th className="px-3 py-2 text-left">IP Address</th>
                        <th className="px-3 py-2 text-left">Owner</th>
                      </tr>
                    </thead>
                    <tbody>
                      {preview.map((a, i) => (
                        <tr key={i} className="border-t">
                          <td className="px-3 py-1.5">{a.name ?? '—'}</td>
                          <td className="px-3 py-1.5">{a.type ?? '—'}</td>
                          <td className="px-3 py-1.5">{a.criticality ?? '—'}</td>
                          <td className="px-3 py-1.5">{a.ip_address ?? '—'}</td>
                          <td className="px-3 py-1.5">{a.owner ?? '—'}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            </>
          )}
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          {preview && (
            <Button type="button" onClick={handleImport} disabled={isPending}>
              <Upload className="mr-1.5 h-4 w-4" />
              {isPending ? 'Importing…' : `Import ${preview.length} Assets`}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
