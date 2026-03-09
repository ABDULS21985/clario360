'use client';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { formatNumber, formatPercentage } from '@/lib/format';
import type { AIValidationResult } from '@/types/ai-governance';

interface ConfusionMatrixProps {
  result: AIValidationResult;
}

function matrixCellClass(kind: 'tp' | 'fp' | 'tn' | 'fn') {
  switch (kind) {
    case 'tp':
    case 'tn':
      return 'border-emerald-200 bg-emerald-50/90';
    case 'fp':
      return 'border-red-200 bg-red-50/90';
    default:
      return 'border-amber-200 bg-amber-50/90';
  }
}

function cellShare(value: number, total: number) {
  return total > 0 ? value / total : 0;
}

export function ConfusionMatrix({ result }: ConfusionMatrixProps) {
  const total = result.dataset_size;
  const cells = [
    { title: 'True Positive', short: 'TP', value: result.true_positives, kind: 'tp' as const },
    { title: 'False Positive', short: 'FP', value: result.false_positives, kind: 'fp' as const },
    { title: 'False Negative', short: 'FN', value: result.false_negatives, kind: 'fn' as const },
    { title: 'True Negative', short: 'TN', value: result.true_negatives, kind: 'tn' as const },
  ];

  return (
    <Card className="border-border/70">
      <CardHeader>
        <CardTitle>Confusion Matrix</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="grid gap-3 sm:grid-cols-2">
          {cells.map((cell) => (
            <div
              key={cell.short}
              className={`rounded-2xl border p-4 ${matrixCellClass(cell.kind)}`}
            >
              <div className="text-xs font-semibold uppercase tracking-[0.22em] text-slate-600">
                {cell.short}
              </div>
              <div className="mt-3 text-3xl font-semibold tracking-[-0.05em] text-slate-950">
                {formatNumber(cell.value)}
              </div>
              <div className="mt-2 text-sm text-slate-600">
                {cell.title} • {formatPercentage(cellShare(cell.value, total), 1)}
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
