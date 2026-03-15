'use client';

import { AlertTriangle, CheckCircle2, Clock3, ShieldAlert } from 'lucide-react';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Spinner } from '@/components/ui/spinner';
import { type ConnectionTestResult } from '@/lib/data-suite';

interface WizardStepTestProps {
  loading: boolean;
  connectionLabel: string;
  result: ConnectionTestResult | null;
  error: string | null;
  onEditConnection: () => void;
  onRetry: () => void;
  onContinueWithoutDetails: () => void;
}

export function WizardStepTest({
  loading,
  connectionLabel,
  result,
  error,
  onEditConnection,
  onRetry,
  onContinueWithoutDetails,
}: WizardStepTestProps) {
  return (
    <div className="space-y-4">
      {loading ? (
        <Card className="border-primary/20 bg-primary/5">
          <CardContent className="flex items-center gap-3 py-8">
            <Spinner />
            <div>
              <p className="font-medium">Testing connection to {connectionLabel}…</p>
              <p className="text-sm text-muted-foreground">The source is being provisioned and verified against the backend connector.</p>
            </div>
          </CardContent>
        </Card>
      ) : null}

      {result?.success ? (
        <div className="space-y-4">
          <Alert className="border-emerald-200 bg-emerald-50">
            <CheckCircle2 className="h-4 w-4 text-emerald-600" />
            <AlertTitle className="text-emerald-700">Connected successfully</AlertTitle>
            <AlertDescription className="text-emerald-700">{result.message}</AlertDescription>
          </Alert>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
            <MetricCard title="Latency" value={`${result.latency_ms}ms`} />
            <MetricCard title="Version" value={result.version || 'Unknown'} />
            <MetricCard title="Permissions" value={result.permissions?.join(', ') || 'Read access confirmed'} />
            <MetricCard title="Warnings" value={`${result.warnings?.length ?? 0}`} />
          </div>
          {(result.warnings ?? []).length > 0 ? (
            <div className="space-y-2">
              {(result.warnings ?? []).map((warning) => (
                <Alert key={warning} className="border-amber-200 bg-amber-50">
                  <AlertTriangle className="h-4 w-4 text-amber-600" />
                  <AlertTitle className="text-amber-700">Warning</AlertTitle>
                  <AlertDescription className="text-amber-700">{warning}</AlertDescription>
                </Alert>
              ))}
            </div>
          ) : null}
        </div>
      ) : null}

      {error ? (
        <Alert className="border-rose-200 bg-rose-50">
          <ShieldAlert className="h-4 w-4 text-rose-600" />
          <AlertTitle className="text-rose-700">Connection failed</AlertTitle>
          <AlertDescription className="space-y-2 text-rose-700">
            <p>{error}</p>
            <p>Check that the service is reachable from the platform network and that the credentials are correct.</p>
            <div className="flex gap-2">
              <Button type="button" variant="outline" onClick={onEditConnection}>
                Edit connection
              </Button>
              <Button type="button" onClick={onRetry}>
                Retry
              </Button>
            </div>
          </AlertDescription>
        </Alert>
      ) : null}

      <Button type="button" variant="ghost" onClick={onContinueWithoutDetails} className="px-0 text-destructive">
        <Clock3 className="mr-1.5 h-4 w-4" />
        Continue without test details
      </Button>
    </div>
  );
}

function MetricCard({ title, value }: { title: string; value: string }) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">{title}</CardTitle>
      </CardHeader>
      <CardContent className="text-sm font-medium">{value}</CardContent>
    </Card>
  );
}
