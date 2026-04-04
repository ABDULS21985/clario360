'use client';

import { useEffect, useState } from 'react';
import { showApiError, showSuccess } from '@/lib/toast';
import { enterpriseApi } from '@/lib/enterprise';
import type { AIPredictionLog } from '@/types/ai-governance';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';

interface FeedbackDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  prediction: AIPredictionLog | null;
  onSaved: () => void;
}

export function FeedbackDialog({ open, onOpenChange, prediction, onSaved }: FeedbackDialogProps) {
  const [correct, setCorrect] = useState(true);
  const [notes, setNotes] = useState('');
  const [correctedOutput, setCorrectedOutput] = useState('');
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!open) {
      setCorrect(true);
      setNotes('');
      setCorrectedOutput('');
    }
  }, [open]);

  const submit = async () => {
    if (!prediction) {
      return;
    }
    try {
      setSaving(true);
      let parsedCorrectedOutput: unknown;
      if (correctedOutput.trim() !== '') {
        parsedCorrectedOutput = JSON.parse(correctedOutput);
      }
      await enterpriseApi.ai.submitFeedback(prediction.id, {
        correct,
        notes,
        corrected_output: parsedCorrectedOutput,
      });
      showSuccess('Prediction feedback recorded.', `Prediction ${prediction.id.slice(0, 8)} updated.`);
      onOpenChange(false);
      onSaved();
    } catch (error) {
      showApiError(error);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>Submit Prediction Feedback</DialogTitle>
          <DialogDescription>
            Mark whether this prediction was correct and capture the expected output when it was not.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-5">
          <div className="rounded-lg bg-muted/30 p-4 text-sm">
            <div className="font-medium">{prediction?.model_slug}</div>
            <div className="mt-1 text-muted-foreground">{prediction?.use_case}</div>
          </div>
          <div className="space-y-2">
            <Label>Was the prediction correct?</Label>
            <RadioGroup value={correct ? 'true' : 'false'} onValueChange={(value) => setCorrect(value === 'true')}>
              <div className="flex items-center space-x-2">
                <RadioGroupItem value="true" id="feedback-true" />
                <Label htmlFor="feedback-true">Correct</Label>
              </div>
              <div className="flex items-center space-x-2">
                <RadioGroupItem value="false" id="feedback-false" />
                <Label htmlFor="feedback-false">Incorrect</Label>
              </div>
            </RadioGroup>
          </div>
          {!correct ? (
            <div className="space-y-2">
              <Label htmlFor="corrected-output">Corrected Output JSON</Label>
              <Textarea
                id="corrected-output"
                value={correctedOutput}
                onChange={(event) => setCorrectedOutput(event.target.value)}
                placeholder='{"expected":"value"}'
                className="min-h-32 font-mono text-xs"
              />
            </div>
          ) : null}
          <div className="space-y-2">
            <Label htmlFor="feedback-notes">Reviewer Notes</Label>
            <Textarea
              id="feedback-notes"
              value={notes}
              onChange={(event) => setNotes(event.target.value)}
              placeholder="Context for auditors and model owners."
              className="min-h-24"
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={() => void submit()} disabled={saving}>
            {saving ? 'Saving…' : 'Save Feedback'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
