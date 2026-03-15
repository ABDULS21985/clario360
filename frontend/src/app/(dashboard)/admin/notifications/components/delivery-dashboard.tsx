'use client';

import { useState } from 'react';
import { useDeliveryStats, useRetryFailedDeliveries } from '@/hooks/use-delivery-stats';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { showSuccess, showApiError } from '@/lib/toast';
import { Send, CheckCircle, XCircle, Clock, TrendingUp, RotateCw } from 'lucide-react';
import { DeliveryCharts } from './delivery-charts';

type Period = '7d' | '30d' | '90d';
type Channel = 'email' | 'in_app' | 'push' | 'webhook' | undefined;

export function DeliveryDashboard() {
  const [period, setPeriod] = useState<Period>('7d');
  const [channel, setChannel] = useState<Channel>(undefined);
  const [retryOpen, setRetryOpen] = useState(false);

  const { data: stats, isLoading, isError, refetch } = useDeliveryStats({
    period,
    channel: channel as 'email' | 'in_app' | 'push' | 'webhook' | undefined,
  });

  const retryMutation = useRetryFailedDeliveries();

  const handleRetry = async () => {
    try {
      const result = await retryMutation.mutateAsync({
        channel: channel ?? undefined,
      });
      showSuccess(`Retrying ${result.retried} failed deliveries`);
      setRetryOpen(false);
      refetch();
    } catch (error) {
      showApiError(error);
    }
  };

  if (isLoading) return <LoadingSkeleton variant="card" count={3} />;
  if (isError) return <ErrorState message="Failed to load delivery statistics" onRetry={() => refetch()} />;
  if (!stats) return null;

  return (
    <div className="space-y-6">
      {/* Controls */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <span className="text-sm text-muted-foreground">Period:</span>
          {(['7d', '30d', '90d'] as Period[]).map((p) => (
            <Button
              key={p}
              variant={period === p ? 'default' : 'outline'}
              size="sm"
              onClick={() => setPeriod(p)}
            >
              {p}
            </Button>
          ))}
        </div>
        <div className="flex items-center gap-2">
          <span className="text-sm text-muted-foreground">Channel:</span>
          {[
            { value: undefined, label: 'All' },
            { value: 'email' as const, label: 'Email' },
            { value: 'in_app' as const, label: 'In-App' },
            { value: 'push' as const, label: 'Push' },
            { value: 'webhook' as const, label: 'Webhook' },
          ].map((ch) => (
            <Button
              key={ch.label}
              variant={channel === ch.value ? 'default' : 'outline'}
              size="sm"
              onClick={() => setChannel(ch.value)}
            >
              {ch.label}
            </Button>
          ))}
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-5">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Sent</CardTitle>
            <Send className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{(stats.total_sent ?? 0).toLocaleString()}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Delivered</CardTitle>
            <CheckCircle className="h-4 w-4 text-green-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{(stats.delivered ?? 0).toLocaleString()}</div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Failed</CardTitle>
            <XCircle className="h-4 w-4 text-destructive" />
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <span className="text-2xl font-bold">{(stats.failed ?? 0).toLocaleString()}</span>
              {(stats.failed ?? 0) > 0 && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setRetryOpen(true)}
                  className="h-7 text-xs"
                >
                  <RotateCw className="mr-1 h-3 w-3" />
                  Retry
                </Button>
              )}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Delivery Rate</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {((stats.delivery_rate ?? 0) * 100).toFixed(1)}%
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Avg Delivery</CardTitle>
            <Clock className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {(stats.avg_delivery_time_ms ?? 0) < 1000
                ? `${stats.avg_delivery_time_ms ?? 0}ms`
                : `${((stats.avg_delivery_time_ms ?? 0) / 1000).toFixed(1)}s`}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Charts */}
      <DeliveryCharts stats={stats} />

      {/* Retry Dialog */}
      <ConfirmDialog
        open={retryOpen}
        onOpenChange={setRetryOpen}
        title="Retry Failed Deliveries"
        description={`This will retry ${stats.failed ?? 0} failed notification deliveries${channel ? ` for the ${channel} channel` : ''}. Continue?`}
        confirmLabel="Retry All"
        onConfirm={handleRetry}
        loading={retryMutation.isPending}
      />
    </div>
  );
}
