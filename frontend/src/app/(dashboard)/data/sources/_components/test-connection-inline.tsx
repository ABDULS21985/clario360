'use client';

import { useEffect, useState } from 'react';
import { CheckCircle2, ExternalLink, ShieldAlert } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Spinner } from '@/components/ui/spinner';
import { type ConnectionTestResult } from '@/lib/data-suite';

interface TestConnectionInlineProps {
  loading: boolean;
  result?: ConnectionTestResult | null;
  error?: string | null;
  onEdit?: () => void;
}

export function TestConnectionInline({
  loading,
  result,
  error,
  onEdit,
}: TestConnectionInlineProps) {
  const [visible, setVisible] = useState(Boolean(loading || result || error));

  useEffect(() => {
    if (loading || result || error) {
      setVisible(true);
    }
  }, [loading, result, error]);

  useEffect(() => {
    if (loading || (!result && !error)) {
      return;
    }
    const timer = window.setTimeout(() => setVisible(false), 10_000);
    return () => window.clearTimeout(timer);
  }, [loading, result, error]);

  if (!visible) {
    return null;
  }

  if (loading) {
    return (
      <div className="mt-3 flex items-center gap-2 rounded-md bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
        <Spinner size="sm" />
        Testing connection…
      </div>
    );
  }

  if (result?.success) {
    return (
      <div className="mt-3 flex items-center gap-2 rounded-md bg-emerald-50 px-3 py-2 text-xs text-emerald-700">
        <CheckCircle2 className="h-3.5 w-3.5" />
        Connected in {result.latency_ms}ms
        {result.version ? ` • ${result.version}` : ''}
      </div>
    );
  }

  if (error) {
    return (
      <div className="mt-3 space-y-2 rounded-md bg-rose-50 px-3 py-2 text-xs text-rose-700">
        <div className="flex items-start gap-2">
          <ShieldAlert className="mt-0.5 h-3.5 w-3.5" />
          <span>{error}</span>
        </div>
        {onEdit ? (
          <Button type="button" variant="link" size="sm" className="h-auto px-0 text-rose-700" onClick={onEdit}>
            Edit connection
            <ExternalLink className="ml-1 h-3 w-3" />
          </Button>
        ) : null}
      </div>
    );
  }

  return null;
}
