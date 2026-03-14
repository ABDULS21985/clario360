'use client';

import { useMemo, useState } from 'react';
import { formatISO, subDays } from 'date-fns';
import { FlaskConical } from 'lucide-react';

import { apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { DetectionRule, DetectionRuleTestResult } from '@/types/cyber';
import { parseApiError } from '@/lib/format';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { ScrollArea } from '@/components/ui/scroll-area';

interface RuleTestDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  rule: DetectionRule | null;
}

export function RuleTestDialog({ open, onOpenChange, rule }: RuleTestDialogProps) {
  const [limit, setLimit] = useState(1000);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<DetectionRuleTestResult | null>(null);

  const requestBody = useMemo(
    () => ({
      date_from: formatISO(subDays(new Date(), 14)),
      limit,
    }),
    [limit],
  );

  async function handleRun() {
    if (!rule) {
      return;
    }
    setRunning(true);
    setError(null);
    setResult(null);
    try {
      const response = await apiPost<{ data: DetectionRuleTestResult }>(API_ENDPOINTS.CYBER_RULE_TEST(rule.id), requestBody);
      setResult(response.data);
    } catch (caughtError) {
      setError(parseApiError(caughtError));
    } finally {
      setRunning(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <FlaskConical className="h-5 w-5 text-emerald-700" />
            Test Rule
          </DialogTitle>
          <DialogDescription>
            Dry-run {rule?.name ?? 'this rule'} against recent security events.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 md:grid-cols-[220px_1fr]">
          <div className="space-y-4 rounded-[22px] border p-4">
            <div className="space-y-2">
              <Label htmlFor="rule-test-limit">Event limit</Label>
              <Input
                id="rule-test-limit"
                type="number"
                min={100}
                max={5000}
                step={100}
                value={limit}
                onChange={(event) => setLimit(Number(event.target.value) || 1000)}
              />
            </div>
            <p className="text-sm text-muted-foreground">
              The backend evaluates the rule against the latest events since {requestBody.date_from}.
            </p>
            <Button className="w-full" onClick={() => void handleRun()} disabled={running || !rule}>
              {running ? 'Running…' : 'Run Test'}
            </Button>
            {error ? (
              <div className="rounded-2xl border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
                {error}
              </div>
            ) : null}
          </div>

          <div className="rounded-[22px] border p-4">
            <div className="mb-4 flex items-center justify-between">
              <div>
                <p className="text-sm font-medium">Results</p>
                <p className="text-sm text-muted-foreground">
                  {result ? `${result.count} match${result.count === 1 ? '' : 'es'} found` : 'Run the test to preview matches.'}
                </p>
              </div>
              {result ? <Badge variant="outline">{result.count} matches</Badge> : null}
            </div>

            <ScrollArea className="h-[420px] pr-3">
              <div className="space-y-3">
                {result?.matches?.length ? (
                  result.matches.map((match, index) => (
                    <div key={`${match.timestamp}-${index}`} className="rounded-2xl border bg-slate-950/95 p-4 text-slate-100">
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <span className="text-sm font-medium">Match #{index + 1}</span>
                        <span className="font-mono text-xs text-slate-300">{new Date(match.timestamp).toLocaleString()}</span>
                      </div>
                      <pre className="mt-3 overflow-x-auto text-xs">
                        {JSON.stringify(match.match_details, null, 2)}
                      </pre>
                    </div>
                  ))
                ) : (
                  <div className="rounded-2xl border border-dashed p-6 text-center text-sm text-muted-foreground">
                    No matches to preview.
                  </div>
                )}
              </div>
            </ScrollArea>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
