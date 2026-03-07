'use client';

import { ShieldCheck } from 'lucide-react';
import { Checkbox } from '@/components/ui/checkbox';
import { Label } from '@/components/ui/label';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { SchemaTree } from '@/app/(dashboard)/data/sources/[id]/_components/schema-tree';
import { type DiscoveredSchema } from '@/lib/data-suite';

interface WizardStepSchemaProps {
  schema: DiscoveredSchema | null;
  loading: boolean;
  error: string | null;
  reviewed: boolean;
  onReviewedChange: (value: boolean) => void;
  onRetry: () => void;
  canViewPii: boolean;
}

export function WizardStepSchema({
  schema,
  loading,
  error,
  reviewed,
  onReviewedChange,
  onRetry,
  canViewPii,
}: WizardStepSchemaProps) {
  if (loading) {
    return <LoadingSkeleton variant="chart" />;
  }

  if (error) {
    return <ErrorState message={error} onRetry={onRetry} />;
  }

  if (!schema) {
    return null;
  }

  return (
    <div className="space-y-4">
      <SchemaTree schema={schema} canViewPii={canViewPii} />

      <Alert className="border-emerald-200 bg-emerald-50">
        <ShieldCheck className="h-4 w-4 text-emerald-600" />
        <AlertTitle className="text-emerald-700">Schema review</AlertTitle>
        <AlertDescription className="text-emerald-700">
          Review the discovered tables and PII flags before continuing. Sensitive columns will remain masked for users without `data:pii`.
        </AlertDescription>
      </Alert>

      <div className="flex items-center gap-3 rounded-lg border p-4">
        <Checkbox checked={reviewed} onCheckedChange={(checked) => onReviewedChange(Boolean(checked))} />
        <Label>I&apos;ve reviewed the schema and PII classifications.</Label>
      </div>
    </div>
  );
}
