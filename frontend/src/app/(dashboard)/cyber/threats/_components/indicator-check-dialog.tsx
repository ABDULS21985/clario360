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
import { apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { CheckCircle, XCircle, Search, AlertCircle } from 'lucide-react';
import { toast } from 'sonner';
import type { IndicatorCheckResult } from '@/types/cyber';
import { getIndicatorTypeLabel } from '@/lib/cyber-threats';

interface IndicatorCheckDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function IndicatorCheckDialog({ open, onOpenChange }: IndicatorCheckDialogProps) {
  const [raw, setRaw] = useState('');
  const [loading, setLoading] = useState(false);
  const [results, setResults] = useState<IndicatorCheckResult[] | null>(null);

  const handleCheck = async () => {
    const indicators = raw
      .split('\n')
      .map((l) => l.trim())
      .filter(Boolean);

    if (indicators.length === 0) return;
    setLoading(true);
    setResults(null);

    try {
      const res = await apiPost<{ data: IndicatorCheckResult[] }>(
        API_ENDPOINTS.CYBER_INDICATORS_CHECK,
        { values: indicators },
      );
      setResults(res.data);
    } catch {
      toast.error('Indicator check failed');
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    setRaw('');
    setResults(null);
    onOpenChange(false);
  };

  const matched = results?.filter((r) => (r.indicators?.length ?? 0) > 0) ?? [];
  const clean = results?.filter((r) => (r.indicators?.length ?? 0) === 0) ?? [];

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Search className="h-5 w-5 text-primary" />
            Indicator Check
          </DialogTitle>
          <DialogDescription>
            Paste IPs, domains, hashes, or URLs (one per line) to check against the threat intelligence database.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div>
            <Label htmlFor="indicators-input">Indicators (one per line)</Label>
            <Textarea
              id="indicators-input"
              value={raw}
              onChange={(e) => setRaw(e.target.value)}
              placeholder="8.8.8.8&#10;malicious-domain.com&#10;d41d8cd98f00b204e9800998ecf8427e"
              rows={5}
              className="mt-1 font-mono text-xs"
            />
          </div>

          {results && (
            <div className="space-y-3">
              {matched.length > 0 && (
                <div className="rounded-xl border border-red-200 bg-red-50/50 p-3 dark:border-red-900 dark:bg-red-950/20">
                  <div className="mb-2 flex items-center gap-2 text-xs font-semibold text-red-700 dark:text-red-400">
                    <AlertCircle className="h-3.5 w-3.5" />
                    {matched.length} Malicious Indicator{matched.length !== 1 ? 's' : ''}
                  </div>
                  <div className="space-y-1.5">
                    {matched.map((r, i) => (
                      <div key={i} className="flex items-center gap-3 text-sm">
                        <XCircle className="h-4 w-4 shrink-0 text-red-600" />
                        <span className="font-mono text-xs flex-1 truncate">{r.value}</span>
                        <div className="flex flex-wrap items-center gap-2">
                          {r.indicators.slice(0, 2).map((indicator) => (
                            <div key={indicator.id} className="flex items-center gap-1.5 rounded-full bg-background px-2 py-1">
                              <span className="text-[11px] text-muted-foreground">{getIndicatorTypeLabel(indicator.type)}</span>
                              <SeverityIndicator severity={indicator.severity} />
                            </div>
                          ))}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
              {clean.length > 0 && (
                <div className="rounded-xl border border-green-200 bg-green-50/50 p-3 dark:border-green-900 dark:bg-green-950/20">
                  <div className="mb-2 flex items-center gap-2 text-xs font-semibold text-green-700 dark:text-green-400">
                    <CheckCircle className="h-3.5 w-3.5" />
                    {clean.length} Clean
                  </div>
                  <div className="space-y-1">
                    {clean.map((r, i) => (
                      <div key={i} className="flex items-center gap-2 text-xs">
                        <CheckCircle className="h-3.5 w-3.5 shrink-0 text-green-600" />
                        <span className="font-mono">{r.value}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={handleClose}>Close</Button>
          <Button
            type="button"
            onClick={handleCheck}
            disabled={!raw.trim() || loading}
          >
            <Search className="mr-1.5 h-3.5 w-3.5" />
            {loading ? 'Checking…' : 'Check Indicators'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
