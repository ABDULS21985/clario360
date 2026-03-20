'use client';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { formatNumber, formatPercentage, titleCase } from '@/lib/format';
import type { AIValidationMetricsSummary } from '@/types/ai-governance';

interface SeverityBreakdownTableProps {
  title?: string;
  label?: string;
  breakdown: Record<string, AIValidationMetricsSummary>;
  order?: string[];
}

const severityOrder = ['critical', 'high', 'medium', 'low', 'unclassified'];

export function SeverityBreakdownTable({
  title = 'Severity Breakdown',
  label = 'Severity',
  breakdown,
  order = severityOrder,
}: SeverityBreakdownTableProps) {
  const entries = Object.entries(breakdown).sort((left, right) => {
    const leftIndex = order.indexOf(left[0]);
    const rightIndex = order.indexOf(right[0]);
    return (leftIndex === -1 ? 99 : leftIndex) - (rightIndex === -1 ? 99 : rightIndex);
  });

  return (
    <Card className="border-border/70">
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{label}</TableHead>
              <TableHead>Precision</TableHead>
              <TableHead>Recall</TableHead>
              <TableHead>F1</TableHead>
              <TableHead>Count</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {entries.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} className="text-sm text-muted-foreground">
                  No breakdown data was recorded for this validation run.
                </TableCell>
              </TableRow>
            ) : null}
            {entries.map(([key, metrics]) => (
              <TableRow key={key}>
                <TableCell className="font-medium">{titleCase(key)}</TableCell>
                <TableCell>{formatPercentage(metrics.precision, 1)}</TableCell>
                <TableCell>{formatPercentage(metrics.recall, 1)}</TableCell>
                <TableCell>{formatPercentage(metrics.f1_score, 1)}</TableCell>
                <TableCell>{formatNumber(metrics.dataset_size)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
