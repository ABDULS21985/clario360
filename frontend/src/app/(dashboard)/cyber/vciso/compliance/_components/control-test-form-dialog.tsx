'use client';

import { useState, useEffect } from 'react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import type {
  VCISOControlTest,
  ControlTestType,
  ControlTestResult,
} from '@/types/cyber';

const TEST_TYPES: { label: string; value: ControlTestType }[] = [
  { label: 'Design', value: 'design' },
  { label: 'Operating Effectiveness', value: 'operating_effectiveness' },
];

const TEST_RESULTS: { label: string; value: ControlTestResult }[] = [
  { label: 'Effective', value: 'effective' },
  { label: 'Partially Effective', value: 'partially_effective' },
  { label: 'Ineffective', value: 'ineffective' },
  { label: 'Not Tested', value: 'not_tested' },
];

interface ControlTestFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  controlTest?: VCISOControlTest | null;
  onSuccess: () => void;
}

export function ControlTestFormDialog({
  open,
  onOpenChange,
  controlTest,
  onSuccess,
}: ControlTestFormDialogProps) {
  const isEditing = !!controlTest;

  const [controlName, setControlName] = useState('');
  const [framework, setFramework] = useState('');
  const [testType, setTestType] = useState<ControlTestType | ''>('');
  const [result, setResult] = useState<ControlTestResult | ''>('');
  const [testerName, setTesterName] = useState('');
  const [findings, setFindings] = useState('');
  const [nextTestDate, setNextTestDate] = useState('');

  useEffect(() => {
    if (open) {
      if (controlTest) {
        setControlName(controlTest.control_name);
        setFramework(controlTest.framework);
        setTestType(controlTest.test_type);
        setResult(controlTest.result);
        setTesterName(controlTest.tester_name);
        setFindings(controlTest.findings);
        setNextTestDate(controlTest.next_test_date?.slice(0, 10) ?? '');
      } else {
        setControlName('');
        setFramework('');
        setTestType('');
        setResult('');
        setTesterName('');
        setFindings('');
        setNextTestDate('');
      }
    }
  }, [open, controlTest]);

  const createMutation = useApiMutation<VCISOControlTest, Record<string, unknown>>(
    'post',
    API_ENDPOINTS.CYBER_VCISO_CONTROL_TESTS,
    {
      invalidateKeys: ['vciso-control-tests'],
      successMessage: 'Test recorded successfully',
      onSuccess: () => {
        onOpenChange(false);
        onSuccess();
      },
    },
  );

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!controlName.trim()) {
      toast.error('Control name is required');
      return;
    }
    if (!framework.trim()) {
      toast.error('Framework is required');
      return;
    }
    if (!testType) {
      toast.error('Test type is required');
      return;
    }
    if (!result) {
      toast.error('Result is required');
      return;
    }
    if (!testerName.trim()) {
      toast.error('Tester name is required');
      return;
    }

    const payload = {
      control_name: controlName.trim(),
      framework: framework.trim(),
      test_type: testType,
      result,
      tester_name: testerName.trim(),
      findings: findings.trim(),
      next_test_date: nextTestDate || undefined,
    };

    createMutation.mutate(payload);
  };

  const isSubmitting = createMutation.isPending;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Record Control Test</DialogTitle>
          <DialogDescription>
            Record a new control effectiveness test result.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-5">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="ct-control-name">Control Name</Label>
              <Input
                id="ct-control-name"
                placeholder="e.g., AC-2 Account Management"
                value={controlName}
                onChange={(e) => setControlName(e.target.value)}
                disabled={isSubmitting}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="ct-framework">Framework</Label>
              <Input
                id="ct-framework"
                placeholder="e.g., NIST 800-53"
                value={framework}
                onChange={(e) => setFramework(e.target.value)}
                disabled={isSubmitting}
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="ct-test-type">Test Type</Label>
              <Select
                value={testType}
                onValueChange={(v) => setTestType(v as ControlTestType)}
                disabled={isSubmitting}
              >
                <SelectTrigger id="ct-test-type">
                  <SelectValue placeholder="Select test type" />
                </SelectTrigger>
                <SelectContent>
                  {TEST_TYPES.map((t) => (
                    <SelectItem key={t.value} value={t.value}>
                      {t.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="ct-result">Result</Label>
              <Select
                value={result}
                onValueChange={(v) => setResult(v as ControlTestResult)}
                disabled={isSubmitting}
              >
                <SelectTrigger id="ct-result">
                  <SelectValue placeholder="Select result" />
                </SelectTrigger>
                <SelectContent>
                  {TEST_RESULTS.map((r) => (
                    <SelectItem key={r.value} value={r.value}>
                      {r.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="ct-tester-name">Tester Name</Label>
              <Input
                id="ct-tester-name"
                placeholder="e.g., Jane Smith"
                value={testerName}
                onChange={(e) => setTesterName(e.target.value)}
                disabled={isSubmitting}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="ct-next-test-date">Next Test Date</Label>
              <Input
                id="ct-next-test-date"
                type="date"
                value={nextTestDate}
                onChange={(e) => setNextTestDate(e.target.value)}
                disabled={isSubmitting}
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="ct-findings">Findings</Label>
            <Textarea
              id="ct-findings"
              placeholder="Describe the test findings..."
              value={findings}
              onChange={(e) => setFindings(e.target.value)}
              disabled={isSubmitting}
              className="min-h-[120px]"
            />
          </div>

          <div className="flex justify-end gap-2 pt-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isSubmitting}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? 'Recording...' : 'Record Test'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
