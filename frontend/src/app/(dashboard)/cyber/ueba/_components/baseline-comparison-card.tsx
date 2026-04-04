'use client';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

export function BaselineComparisonCard({
  title,
  expected,
  actual,
}: {
  title: string;
  expected: unknown;
  actual: unknown;
}) {
  return (
    <Card className="h-full">
      <CardHeader className="pb-3">
        <CardTitle className="text-sm">{title}</CardTitle>
      </CardHeader>
      <CardContent className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <div className="rounded-lg border bg-muted/30 p-3">
          <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">Expected</div>
          <pre className="overflow-auto text-xs">{JSON.stringify(expected, null, 2)}</pre>
        </div>
        <div className="rounded-lg border bg-muted/30 p-3">
          <div className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">Actual</div>
          <pre className="overflow-auto text-xs">{JSON.stringify(actual, null, 2)}</pre>
        </div>
      </CardContent>
    </Card>
  );
}
