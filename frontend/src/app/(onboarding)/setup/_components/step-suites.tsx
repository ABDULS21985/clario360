'use client';

import { useEffect, useState } from 'react';
import { AlertCircle, ChevronLeft, Loader2 } from 'lucide-react';

import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { isApiError } from '@/types/api';

import { SUITES } from './shared';
import { SuiteSelectorCard } from './suite-selector-card';

export function StepSuites({
  initialSelected,
  onBack,
  onSaved,
  onPersist,
}: {
  initialSelected: string[];
  onBack: () => void;
  onSaved: () => Promise<void>;
  onPersist: (selected: string[]) => void;
}) {
  const [selected, setSelected] = useState<string[]>(initialSelected.length > 0 ? initialSelected : ['cyber', 'data', 'visus']);
  const [apiError, setApiError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    onPersist(selected);
  }, [selected, onPersist]);

  const toggleSuite = (suiteId: string) => {
    setSelected((current) =>
      current.includes(suiteId) ? current.filter((item) => item !== suiteId) : [...current, suiteId],
    );
  };

  const submit = async () => {
    if (selected.length === 0) {
      setApiError('Select at least one suite.');
      return;
    }

    setApiError(null);
    setIsSubmitting(true);
    try {
      await apiPost(API_ENDPOINTS.ONBOARDING_SUITES, { active_suites: selected });
      await onSaved();
    } catch (error) {
      setApiError(isApiError(error) ? error.message : 'Failed to save suite selection.');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="space-y-6">
      {apiError ? (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{apiError}</AlertDescription>
        </Alert>
      ) : null}

      <div className="grid gap-4 md:grid-cols-2">
        {SUITES.map((suite) => (
          <SuiteSelectorCard
            key={suite.id}
            suite={suite}
            active={selected.includes(suite.id)}
            onToggle={() => toggleSuite(suite.id)}
          />
        ))}
      </div>

      <div className="flex justify-between">
        <Button type="button" variant="outline" onClick={onBack}>
          <ChevronLeft className="mr-1 h-4 w-4" />
          Back
        </Button>
        <Button type="button" disabled={isSubmitting || selected.length === 0} onClick={() => void submit()}>
          {isSubmitting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
          Continue
        </Button>
      </div>
    </div>
  );
}
