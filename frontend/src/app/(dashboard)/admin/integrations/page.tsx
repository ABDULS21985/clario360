"use client";

import { useEffect, useState } from "react";
import { AlertTriangle, CheckCircle2, Clock3, Plug, RefreshCw, RotateCcw, TestTube2 } from "lucide-react";
import { toast } from "sonner";
import { PageHeader } from "@/components/common/page-header";
import { PermissionRedirect } from "@/components/common/permission-redirect";
import { EmptyState } from "@/components/common/empty-state";
import { RelativeTime } from "@/components/shared/relative-time";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { apiGet, apiPost } from "@/lib/api";
import type { PaginatedResponse } from "@/types/api";
import type { IntegrationRecord } from "@/types/integration";

async function fetchIntegrations(): Promise<IntegrationRecord[]> {
  const response = await apiGet<PaginatedResponse<IntegrationRecord>>("/api/v1/integrations", {
    page: 1,
    per_page: 100,
    sort: "updated_at",
    order: "desc",
  });
  return response.data ?? [];
}

function statusBadgeVariant(status: IntegrationRecord["status"]): "default" | "secondary" | "destructive" | "outline" {
  switch (status) {
    case "active":
      return "default";
    case "error":
      return "destructive";
    case "setup_pending":
      return "secondary";
    default:
      return "outline";
  }
}

function prettyType(type: IntegrationRecord["type"]): string {
  switch (type) {
    case "servicenow":
      return "ServiceNow";
    default:
      return type.charAt(0).toUpperCase() + type.slice(1);
  }
}

export default function AdminIntegrationsPage() {
  const [items, setItems] = useState<IntegrationRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);

  const load = async () => {
    setLoading(true);
    setError(null);
    try {
      setItems(await fetchIntegrations());
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load integrations");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
  }, []);

  const handleTest = async (integrationId: string) => {
    setBusyId(integrationId);
    try {
      const response = await apiPost<{ data: { response_code: number; success: boolean } }>(`/api/v1/integrations/${integrationId}/test`);
      toast.success(`Test completed with HTTP ${response.data.response_code}`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Integration test failed");
    } finally {
      setBusyId(null);
    }
  };

  const handleRetry = async (integrationId: string) => {
    setBusyId(integrationId);
    try {
      const response = await apiPost<{ data: { retried_count: number } }>(`/api/v1/integrations/${integrationId}/retry-failed`);
      toast.success(`Re-queued ${response.data.retried_count} failed deliveries`);
      await load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Retry failed");
    } finally {
      setBusyId(null);
    }
  };

  return (
    <PermissionRedirect permission="tenant:write">
      <div className="space-y-6">
        <PageHeader
          title="External Integrations"
          description="Slack, Teams, Jira, ServiceNow, and generic webhook connections for operational workflows."
          actions={
            <Button variant="outline" onClick={() => void load()} disabled={loading}>
              <RefreshCw className={`mr-2 h-4 w-4 ${loading ? "animate-spin" : ""}`} />
              Refresh
            </Button>
          }
        />

        {error ? (
          <Alert variant="destructive">
            <AlertTriangle className="h-4 w-4" />
            <AlertTitle>Unable to load integrations</AlertTitle>
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        ) : null}

        {loading ? (
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            {Array.from({ length: 6 }).map((_, index) => (
              <Card key={index}>
                <CardHeader className="space-y-3">
                  <Skeleton className="h-5 w-40" />
                  <Skeleton className="h-4 w-28" />
                </CardHeader>
                <CardContent className="space-y-3">
                  <Skeleton className="h-4 w-full" />
                  <Skeleton className="h-4 w-2/3" />
                  <Skeleton className="h-9 w-full" />
                </CardContent>
              </Card>
            ))}
          </div>
        ) : items.length === 0 ? (
          <EmptyState
            icon={Plug}
            title="No integrations configured"
            description="The backend management API is live. Create Slack, Teams, Jira, ServiceNow, or webhook integrations through the integration endpoints or upcoming setup wizard."
          />
        ) : (
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            {items.map((item) => (
              <Card key={item.id} className="border-border/70">
                <CardHeader className="space-y-3">
                  <div className="flex items-start justify-between gap-3">
                    <div className="space-y-1">
                      <CardTitle className="text-base">{item.name}</CardTitle>
                      <CardDescription>{prettyType(item.type)}</CardDescription>
                    </div>
                    <Badge variant={statusBadgeVariant(item.status)}>{item.status.replace("_", " ")}</Badge>
                  </div>
                  {item.description ? (
                    <p className="text-sm text-muted-foreground">{item.description}</p>
                  ) : null}
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid gap-3 text-sm">
                    <div className="flex items-center justify-between gap-3">
                      <span className="text-muted-foreground">Deliveries</span>
                      <span className="font-medium">{item.delivery_count.toLocaleString()}</span>
                    </div>
                    <div className="flex items-center justify-between gap-3">
                      <span className="text-muted-foreground">Errors</span>
                      <span className="font-medium">{item.error_count}</span>
                    </div>
                    <div className="flex items-center justify-between gap-3">
                      <span className="text-muted-foreground">Last used</span>
                      <span className="text-right text-xs">
                        {item.last_used_at ? <RelativeTime date={item.last_used_at} /> : "Never"}
                      </span>
                    </div>
                    <div className="flex items-center justify-between gap-3">
                      <span className="text-muted-foreground">Updated</span>
                      <span className="text-right text-xs">
                        <RelativeTime date={item.updated_at} />
                      </span>
                    </div>
                  </div>

                  {item.error_message ? (
                    <Alert variant={item.status === "error" ? "destructive" : "default"}>
                      {item.status === "error" ? (
                        <AlertTriangle className="h-4 w-4" />
                      ) : (
                        <Clock3 className="h-4 w-4" />
                      )}
                      <AlertTitle>{item.status === "error" ? "Delivery issue" : "Attention needed"}</AlertTitle>
                      <AlertDescription>{item.error_message}</AlertDescription>
                    </Alert>
                  ) : (
                    <div className="flex items-center gap-2 rounded-md border border-emerald-500/20 bg-emerald-500/5 px-3 py-2 text-sm text-emerald-700 dark:text-emerald-300">
                      <CheckCircle2 className="h-4 w-4" />
                      No current integration error recorded.
                    </div>
                  )}

                  <div className="flex flex-wrap gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => void handleTest(item.id)}
                      disabled={busyId === item.id}
                    >
                      <TestTube2 className="mr-2 h-4 w-4" />
                      Test
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => void handleRetry(item.id)}
                      disabled={busyId === item.id}
                    >
                      <RotateCcw className="mr-2 h-4 w-4" />
                      Retry Failed
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </div>
    </PermissionRedirect>
  );
}

