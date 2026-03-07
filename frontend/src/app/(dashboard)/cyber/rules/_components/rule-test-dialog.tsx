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
import { FlaskConical, CheckCircle, XCircle } from 'lucide-react';
import { toast } from 'sonner';
import type { DetectionRule } from '@/types/cyber';

interface TestResult {
  matched: boolean;
  matches: number;
  events_tested: number;
  sample_matches?: Record<string, unknown>[];
  error?: string;
}

interface RuleTestDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  rule: DetectionRule;
}

const SAMPLE_LOG = JSON.stringify({
  event_type: 'process_creation',
  process_name: 'powershell.exe',
  command_line: 'powershell -enc SGVsbG8gV29ybGQ=',
  user: 'SYSTEM',
  timestamp: new Date().toISOString(),
}, null, 2);

export function RuleTestDialog({ open, onOpenChange, rule }: RuleTestDialogProps) {
  const [payload, setPayload] = useState(SAMPLE_LOG);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<TestResult | null>(null);

  const handleTest = async () => {
    setLoading(true);
    setResult(null);
    try {
      const res = await apiPost<{ data: TestResult }>(
        `${API_ENDPOINTS.CYBER_RULES}/${rule.id}/test`,
        { events: [JSON.parse(payload)] },
      );
      setResult(res.data);
    } catch (e) {
      toast.error('Test failed — check your event payload JSON');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <FlaskConical className="h-5 w-5 text-blue-500" />
            Test Rule: {rule.name}
          </DialogTitle>
          <DialogDescription>
            Paste a sample log event (JSON) to test if this rule would trigger.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div>
            <Label htmlFor="test-payload">Sample Event (JSON)</Label>
            <Textarea
              id="test-payload"
              value={payload}
              onChange={(e) => setPayload(e.target.value)}
              rows={8}
              className="mt-1 font-mono text-xs"
            />
          </div>

          {result && (
            <div className={`rounded-xl border p-4 ${result.matched ? 'border-red-200 bg-red-50/50 dark:border-red-900 dark:bg-red-950/20' : 'border-green-200 bg-green-50/50 dark:border-green-900 dark:bg-green-950/20'}`}>
              <div className="flex items-center gap-2 mb-2">
                {result.matched ? (
                  <XCircle className="h-5 w-5 text-red-600" />
                ) : (
                  <CheckCircle className="h-5 w-5 text-green-600" />
                )}
                <span className={`font-semibold ${result.matched ? 'text-red-700 dark:text-red-400' : 'text-green-700 dark:text-green-400'}`}>
                  {result.matched ? `Rule MATCHED (${result.matches} match${result.matches !== 1 ? 'es' : ''})` : 'No match — rule would not trigger'}
                </span>
              </div>
              {result.error && <p className="text-xs text-destructive">{result.error}</p>}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Close</Button>
          <Button type="button" onClick={handleTest} disabled={loading || !payload.trim()}>
            <FlaskConical className="mr-1.5 h-3.5 w-3.5" />
            {loading ? 'Testing…' : 'Run Test'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
