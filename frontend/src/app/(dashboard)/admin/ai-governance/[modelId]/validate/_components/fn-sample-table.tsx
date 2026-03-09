'use client';

import { Fragment, useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { formatPercentage, titleCase, truncate } from '@/lib/format';
import type { AIValidationPredictionSample } from '@/types/ai-governance';

interface FNSampleTableProps {
  samples: AIValidationPredictionSample[];
}

export function FNSampleTable({ samples }: FNSampleTableProps) {
  const [expandedRow, setExpandedRow] = useState<string | null>(null);

  return (
    <Card className="border-border/70">
      <CardHeader>
        <CardTitle>False Negative Samples</CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Event</TableHead>
              <TableHead>Predicted</TableHead>
              <TableHead>Actual</TableHead>
              <TableHead>Confidence</TableHead>
              <TableHead>Rule</TableHead>
              <TableHead>Explanation</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {samples.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="text-sm text-muted-foreground">
                  No false negatives were captured in this validation run.
                </TableCell>
              </TableRow>
            ) : null}
            {samples.map((sample) => {
              const key = sample.prediction_id ?? sample.input_hash;
              const open = expandedRow === key;
              return (
                <Fragment key={key}>
                  <TableRow className="cursor-pointer" onClick={() => setExpandedRow(open ? null : key)}>
                    <TableCell className="font-medium">
                      <div>{sample.input_hash.slice(0, 12)}...</div>
                      <div className="mt-1 text-xs text-muted-foreground">
                        {titleCase(sample.severity || 'unclassified')}
                      </div>
                    </TableCell>
                    <TableCell>{titleCase(sample.predicted_label)}</TableCell>
                    <TableCell>{titleCase(sample.expected_label)}</TableCell>
                    <TableCell>{formatPercentage(sample.confidence, 1)}</TableCell>
                    <TableCell>{sample.rule_type || 'Unknown'}</TableCell>
                    <TableCell className="max-w-[320px] text-muted-foreground">
                      {truncate(sample.explanation || 'No explanation available.', 110)}
                    </TableCell>
                  </TableRow>
                  {open ? (
                    <TableRow>
                      <TableCell colSpan={6} className="bg-slate-50/80">
                        <div className="grid gap-4 lg:grid-cols-2">
                          <div className="space-y-2 text-sm text-slate-700">
                            <div className="font-medium text-slate-900">Full Explanation</div>
                            <p>{sample.explanation || 'No explanation available.'}</p>
                          </div>
                          <div className="space-y-3 text-sm text-slate-700">
                            <div>
                              <div className="font-medium text-slate-900">Predicted Output</div>
                              <pre className="mt-2 overflow-auto rounded-xl bg-white p-3 text-xs text-slate-700">
                                {JSON.stringify(sample.predicted_output, null, 2)}
                              </pre>
                            </div>
                            <div>
                              <div className="font-medium text-slate-900">Event Details</div>
                              <pre className="mt-2 overflow-auto rounded-xl bg-white p-3 text-xs text-slate-700">
                                {JSON.stringify(sample.input_summary, null, 2)}
                              </pre>
                            </div>
                          </div>
                        </div>
                      </TableCell>
                    </TableRow>
                  ) : null}
                </Fragment>
              );
            })}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
