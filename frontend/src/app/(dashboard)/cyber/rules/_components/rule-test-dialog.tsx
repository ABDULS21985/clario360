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
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { FlaskConical, CheckCircle, XCircle, AlertTriangle } from 'lucide-react';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { toast } from 'sonner';
import type { DetectionRule, CyberSeverity } from '@/types/cyber';

interface SampleMatch {
  severity: CyberSeverity;
  title: string;
  event_details?: Record<string, unknown>;
  matched_at?: string;
}

interface TestResult {
  match_count: number;
  hours_tested: number;
  sample_matches: SampleMatch[];
  error?: string;
}

interface RuleTestDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  rule: DetectionRule;
}

export function RuleTestDialog({ open, onOpenChange, rule }: RuleTestDialogProps) {
  const [hours, setHours] = useState(24);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<TestResult | null>(null);

  const handleTest = async () => {
    setLoading(true);
    setResult(null);
    try {
      const res = await apiPost<{ data: TestResult }>(
        `${API_ENDPOINTS.CYBER_RULES}/${rule.id}/test`,
        { hours },
      );
      setResult(res.data);
    } catch {
      toast.error('Test failed');
    } finally {
      setLoading(false);
    }
  };

  const matchCount = result?.match_count ?? 0;
  const highMatch = matchCount > 100;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <FlaskConical className="h-5 w-5 text-blue-500" />
            Test Rule: {rule.name}
          </DialogTitle>
          <DialogDescription>
            Dry-run this rule against historical events to preview matches.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="flex items-center gap-3">
            <Label htmlFor="test-hours" className="shrink-0 text-sm">
              Test against last
            </Label>
            <Input
              id="test-hours"
              type="number"
              min={1}
              max={168}
              value={hours}
              onChange={(e) => setHours(parseInt(e.target.value) || 24)}
              className="w-24"
            />
            <span className="text-sm text-muted-foreground">hours of events</span>
          </div>

          {loading && (
            <div className="space-y-1.5">
              <Progress value={undefined} className="h-1.5" />
              <p className="text-xs text-muted-foreground">
                Testing against {hours}h of historical events…
              </p>
            </div>
          )}

          {result && (
            <div className="space-y-3">
              {/* Summary */}
              <div
                className={`rounded-xl border p-4 ${
                  result.error
                    ? 'border-destructive/30 bg-destructive/5'
                    : matchCount === 0
                    ? 'border-green-200 bg-green-50/50 dark:border-green-900 dark:bg-green-950/20'
                    : 'border-orange-200 bg-orange-50/50 dark:border-orange-900 dark:bg-orange-950/20'
                }`}
              >
                {result.error ? (
                  <div className="flex items-center gap-2">
                    <XCircle className="h-5 w-5 text-destructive" />
                    <span className="text-sm text-destructive">{result.error}</span>
                  </div>
                ) : matchCount === 0 ? (
                  <div className="flex items-center gap-2">
                    <CheckCircle className="h-5 w-5 text-green-600" />
                    <span className="text-sm text-green-700 dark:text-green-400">
                      No matches found. Consider adjusting your rule conditions.
                    </span>
                  </div>
                ) : (
                  <div>
                    <div className="flex items-center gap-2">
                      <FlaskConical className="h-5 w-5 text-orange-600" />
                      <span className="font-semibold text-sm">
                        Would have generated{' '}
                        <strong>{matchCount.toLocaleString()}</strong> alert
                        {matchCount !== 1 ? 's' : ''} in the last {result.hours_tested} hours.
                      </span>
                    </div>
                    {highMatch && (
                      <div className="mt-2 flex items-center gap-1.5 text-xs text-orange-700 dark:text-orange-400">
                        <AlertTriangle className="h-3.5 w-3.5" />
                        This rule may generate excessive alerts. Consider adding filters.
                      </div>
                    )}
                  </div>
                )}
              </div>

              {/* Sample matches */}
              {result.sample_matches && result.sample_matches.length > 0 && (
                <div>
                  <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                    Sample Matches (first {result.sample_matches.length})
                  </p>
                  <div className="space-y-1.5 max-h-48 overflow-y-auto rounded-lg border p-2">
                    {result.sample_matches.map((m, i) => (
                      <div
                        key={i}
                        className="flex items-start justify-between gap-2 rounded-md bg-muted/30 px-2 py-1.5"
                      >
                        <div className="flex items-center gap-2 min-w-0">
                          <SeverityIndicator severity={m.severity} />
                          <span className="truncate text-xs font-medium">{m.title}</span>
                        </div>
                        {m.matched_at && (
                          <span className="shrink-0 text-[10px] text-muted-foreground">
                            {new Date(m.matched_at).toLocaleTimeString()}
                          </span>
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
          <Button type="button" onClick={() => void handleTest()} disabled={loading}>
            <FlaskConical className="mr-1.5 h-3.5 w-3.5" />
            {loading ? 'Testing…' : 'Run Test'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
