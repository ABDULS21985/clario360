'use client';

import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { MessageSquarePlus } from 'lucide-react';
import { enterpriseApi } from '@/lib/enterprise';
import type { AIExplanation, AIPredictionLog } from '@/types/ai-governance';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { RelativeTime } from '@/components/shared/relative-time';
import { ExplanationViewer } from './explanation-viewer';
import { FeedbackDialog } from './feedback-dialog';

interface PredictionLogTableProps {
  modelId: string;
}

export function PredictionLogTable({ modelId }: PredictionLogTableProps) {
  const [page, setPage] = useState(1);
  const [selectedPrediction, setSelectedPrediction] = useState<AIPredictionLog | null>(null);
  const [feedbackPrediction, setFeedbackPrediction] = useState<AIPredictionLog | null>(null);

  const predictionsQuery = useQuery({
    queryKey: ['ai-predictions', modelId, page],
    queryFn: () =>
      enterpriseApi.ai.listPredictions({
        page,
        per_page: 10,
        order: 'desc',
        sort: 'created_at',
        filters: { model_id: modelId },
      }),
    enabled: Boolean(modelId),
  });

  const explanationQuery = useQuery<AIExplanation>({
    queryKey: ['ai-explanation', selectedPrediction?.id],
    queryFn: () => enterpriseApi.ai.getExplanation(selectedPrediction!.id),
    enabled: Boolean(selectedPrediction?.id),
  });

  const meta = predictionsQuery.data?.meta;
  const predictions = predictionsQuery.data?.data ?? [];

  const selectedExplanation = useMemo(() => explanationQuery.data ?? null, [explanationQuery.data]);

  return (
    <>
      <Card className="border-border/70">
        <CardHeader>
          <CardTitle>Prediction Log</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {predictionsQuery.isLoading ? (
            <LoadingSkeleton variant="table-row" count={5} />
          ) : predictionsQuery.error ? (
            <ErrorState
              message="Failed to load prediction logs for this model."
              onRetry={() => void predictionsQuery.refetch()}
            />
          ) : (
            <>
              <div className="rounded-xl border border-border/70">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>When</TableHead>
                      <TableHead>Use Case</TableHead>
                      <TableHead>Version</TableHead>
                      <TableHead>Confidence</TableHead>
                      <TableHead>Latency</TableHead>
                      <TableHead>Feedback</TableHead>
                      <TableHead />
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {predictions.map((prediction) => (
                      <TableRow key={prediction.id}>
                        <TableCell><RelativeTime date={prediction.created_at} /></TableCell>
                        <TableCell className="font-medium">{prediction.use_case}</TableCell>
                        <TableCell>v{prediction.model_version_number ?? 'n/a'}</TableCell>
                        <TableCell>{prediction.confidence ? `${Math.round(prediction.confidence * 100)}%` : 'n/a'}</TableCell>
                        <TableCell>{prediction.latency_ms} ms</TableCell>
                        <TableCell>
                          {prediction.feedback_correct == null ? (
                            <Badge variant="outline">Pending</Badge>
                          ) : prediction.feedback_correct ? (
                            <Badge variant="success">Correct</Badge>
                          ) : (
                            <Badge variant="destructive">Incorrect</Badge>
                          )}
                        </TableCell>
                        <TableCell>
                          <div className="flex justify-end gap-2">
                            <Button variant="ghost" size="sm" onClick={() => setSelectedPrediction(prediction)}>
                              Explain
                            </Button>
                            <Button variant="outline" size="sm" onClick={() => setFeedbackPrediction(prediction)}>
                              <MessageSquarePlus className="mr-1.5 h-3.5 w-3.5" />
                              Feedback
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                    {predictions.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={7} className="py-10 text-center text-sm text-muted-foreground">
                          No prediction logs found for this model.
                        </TableCell>
                      </TableRow>
                    ) : null}
                  </TableBody>
                </Table>
              </div>
              <div className="flex items-center justify-between">
                <div className="text-sm text-muted-foreground">
                  Page {meta?.page ?? 1} of {meta?.total_pages ?? 1}
                </div>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={(meta?.page ?? 1) <= 1}
                    onClick={() => setPage((value) => Math.max(1, value - 1))}
                  >
                    Previous
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={meta ? meta.page >= meta.total_pages : true}
                    onClick={() => setPage((value) => value + 1)}
                  >
                    Next
                  </Button>
                </div>
              </div>
            </>
          )}
        </CardContent>
      </Card>

      <ExplanationViewer explanation={selectedExplanation} />

      <FeedbackDialog
        open={Boolean(feedbackPrediction)}
        onOpenChange={(open) => {
          if (!open) {
            setFeedbackPrediction(null);
          }
        }}
        prediction={feedbackPrediction}
        onSaved={() => {
          void predictionsQuery.refetch();
          if (selectedPrediction?.id) {
            void explanationQuery.refetch();
          }
        }}
      />
    </>
  );
}
