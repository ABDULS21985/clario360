"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import {
  AlertTriangle,
  CheckCircle2,
  ExternalLink,
  Plug,
  RefreshCw,
  RotateCcw,
  Settings2,
  TestTube2,
  ToggleLeft,
  Trash2,
} from "lucide-react";
import { toast } from "sonner";
import { PageHeader } from "@/components/common/page-header";
import { PermissionRedirect } from "@/components/common/permission-redirect";
import { EmptyState } from "@/components/common/empty-state";
import { RelativeTime } from "@/components/shared/relative-time";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { apiDelete, apiPost, apiPut } from "@/lib/api";
import type { ApiResponse } from "@/types/api";
import type { IntegrationProviderStatus, IntegrationRecord, IntegrationStatus, IntegrationType } from "@/types/integration";
import { IntegrationFormDialog } from "./_components/integration-form-dialog";
import {
  fetchIntegrations,
  fetchProviders,
  prepareOAuthInstall,
  prettyType,
  statusBadgeVariant,
  summarizeIntegrationConfig,
} from "./_components/integration-utils";

export default function AdminIntegrationsPage() {
  const [providers, setProviders] = useState<IntegrationProviderStatus[]>([]);
  const [items, setItems] = useState<IntegrationRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busyKey, setBusyKey] = useState<string | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [dialogType, setDialogType] = useState<IntegrationType | null>(null);
  const [editing, setEditing] = useState<IntegrationRecord | null>(null);
  const [deleteCandidate, setDeleteCandidate] = useState<IntegrationRecord | null>(null);

  const countsByType = useMemo(() => {
    return items.reduce<Record<string, number>>((acc, item) => {
      acc[item.type] = (acc[item.type] ?? 0) + 1;
      return acc;
    }, {});
  }, [items]);

  const load = async () => {
    setLoading(true);
    setError(null);
    try {
      const [providerData, integrationData] = await Promise.all([fetchProviders(), fetchIntegrations()]);
      setProviders(providerData);
      setItems(integrationData);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load integrations");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
  }, []);

  const handleSaved = (integration: IntegrationRecord) => {
    setItems((current) => {
      const existing = current.find((item) => item.id === integration.id);
      if (existing) {
        return current.map((item) => (item.id === integration.id ? integration : item));
      }
      return [integration, ...current];
    });
    setEditing(null);
    setDialogType(null);
  };

  const handleTest = async (integrationId: string) => {
    setBusyKey(`test:${integrationId}`);
    try {
      const response = await apiPost<ApiResponse<{ response_code: number; success: boolean }>>(`/api/v1/integrations/${integrationId}/test`);
      toast.success(`Test completed with HTTP ${response.data.response_code}`);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Integration test failed");
    } finally {
      setBusyKey(null);
    }
  };

  const handleRetry = async (integrationId: string) => {
    setBusyKey(`retry:${integrationId}`);
    try {
      const response = await apiPost<ApiResponse<{ retried_count: number }>>(`/api/v1/integrations/${integrationId}/retry-failed`);
      toast.success(`Re-queued ${response.data.retried_count} failed deliveries`);
      await load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Retry failed");
    } finally {
      setBusyKey(null);
    }
  };

  const handleStatus = async (integrationId: string, nextStatus: Exclude<IntegrationStatus, "setup_pending">) => {
    setBusyKey(`status:${integrationId}`);
    try {
      await apiPut<ApiResponse<{ status: string }>>(`/api/v1/integrations/${integrationId}/status`, { status: nextStatus });
      toast.success(`Integration marked ${nextStatus}`);
      await load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Status update failed");
    } finally {
      setBusyKey(null);
    }
  };

  const handleOAuthInstall = async (provider: IntegrationProviderStatus) => {
    if (provider.type !== "slack" && provider.type !== "jira") {
      toast.error("OAuth is not available for this provider");
      return;
    }
    setBusyKey(`oauth:${provider.type}`);
    try {
      const url = await prepareOAuthInstall(provider.type);
      window.location.assign(url);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Unable to start OAuth setup");
    } finally {
      setBusyKey(null);
    }
  };

  const handleDelete = async (integrationId: string) => {
    setBusyKey(`delete:${integrationId}`);
    try {
      await apiDelete(`/api/v1/integrations/${integrationId}`);
      setItems((current) => current.filter((item) => item.id !== integrationId));
      toast.success("Integration deleted");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Delete failed");
      throw err;
    } finally {
      setBusyKey(null);
    }
  };

  return (
    <PermissionRedirect permission="tenant:write">
      <div className="space-y-6">
        <PageHeader
          title="External Integrations"
          description="Operate Slack, Teams, Jira, ServiceNow, and webhook connectors from one place."
          actions={
            <div className="flex gap-2">
              <Button
                variant="outline"
                onClick={() => {
                  setEditing(null);
                  setDialogType("webhook");
                  setDialogOpen(true);
                }}
              >
                <Settings2 className="mr-2 h-4 w-4" />
                Manual Setup
              </Button>
              <Button variant="outline" onClick={() => void load()} disabled={loading}>
                <RefreshCw className={`mr-2 h-4 w-4 ${loading ? "animate-spin" : ""}`} />
                Refresh
              </Button>
            </div>
          }
        />

        {error ? (
          <Alert variant="destructive">
            <AlertTriangle className="h-4 w-4" />
            <AlertTitle>Unable to load integration admin data</AlertTitle>
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        ) : null}

        <section className="space-y-4">
          <div>
            <h2 className="text-lg font-semibold">Provider Readiness</h2>
            <p className="text-sm text-muted-foreground">
              OAuth-backed providers expose their runtime readiness here. Manual setup remains available for every integration type.
            </p>
          </div>

          {loading ? (
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
              {Array.from({ length: 5 }).map((_, index) => (
                <Card key={index}>
                  <CardHeader className="space-y-3">
                    <Skeleton className="h-5 w-36" />
                    <Skeleton className="h-4 w-full" />
                  </CardHeader>
                  <CardContent className="space-y-3">
                    <Skeleton className="h-10 w-full" />
                    <Skeleton className="h-10 w-full" />
                  </CardContent>
                </Card>
              ))}
            </div>
          ) : (
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
              {providers.map((provider) => (
                <Card key={provider.type}>
                  <CardHeader className="space-y-3">
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <CardTitle className="text-base">{provider.name}</CardTitle>
                        <CardDescription>{provider.description}</CardDescription>
                      </div>
                      <Badge variant={provider.configured ? "default" : "outline"}>
                        {provider.configured ? "ready" : "runtime config needed"}
                      </Badge>
                    </div>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="grid gap-2 text-sm">
                      <div className="flex items-center justify-between">
                        <span className="text-muted-foreground">Setup mode</span>
                        <span>{provider.setup_mode}</span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-muted-foreground">Inbound</span>
                        <span>{provider.supports_inbound ? "Yes" : "No"}</span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-muted-foreground">Outbound</span>
                        <span>{provider.supports_outbound ? "Yes" : "No"}</span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-muted-foreground">Configured integrations</span>
                        <span>{countsByType[provider.type] ?? 0}</span>
                      </div>
                    </div>

                    {!provider.configured && provider.missing_config?.length ? (
                      <Alert>
                        <AlertTriangle className="h-4 w-4" />
                        <AlertTitle>Runtime values missing</AlertTitle>
                        <AlertDescription>{provider.missing_config.join(", ")}</AlertDescription>
                      </Alert>
                    ) : null}

                    <div className="flex flex-wrap gap-2">
                      {provider.oauth_enabled ? (
                        <Button
                          variant="outline"
                          disabled={!provider.configured || busyKey === `oauth:${provider.type}`}
                          onClick={() => void handleOAuthInstall(provider)}
                        >
                          <ExternalLink className="mr-2 h-4 w-4" />
                          Connect via OAuth
                        </Button>
                      ) : null}
                      <Button
                        variant="outline"
                        onClick={() => {
                          setEditing(null);
                          setDialogType(provider.type);
                          setDialogOpen(true);
                        }}
                      >
                        <Settings2 className="mr-2 h-4 w-4" />
                        Advanced Setup
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </section>

        <section className="space-y-4">
          <div>
            <h2 className="text-lg font-semibold">Configured Integrations</h2>
            <p className="text-sm text-muted-foreground">
              Inspect delivery state, finish setup-pending installs, and jump into the detail pages for logs and ticket sync.
            </p>
          </div>

          {loading ? (
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
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
              description="Start from the provider cards above or open the manual setup dialog to create the first integration."
            />
          ) : (
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
              {items.map((item) => {
                const summary = summarizeIntegrationConfig(item);
                const nextStatus = item.status === "active" ? "inactive" : "active";
                const statusActionLabel = item.status === "active" ? "Disable" : "Enable";
                return (
                  <Card key={item.id} className="border-border/70">
                    <CardHeader className="space-y-3">
                      <div className="flex items-start justify-between gap-3">
                        <div className="space-y-1">
                          <CardTitle className="text-base">{item.name}</CardTitle>
                          <CardDescription>{prettyType(item.type)}</CardDescription>
                        </div>
                        <Badge variant={statusBadgeVariant(item.status)}>{item.status.replace("_", " ")}</Badge>
                      </div>
                      {item.description ? <p className="text-sm text-muted-foreground">{item.description}</p> : null}
                    </CardHeader>
                    <CardContent className="space-y-4">
                      <div className="grid gap-2 text-sm">
                        <div className="flex items-center justify-between">
                          <span className="text-muted-foreground">Deliveries</span>
                          <span className="font-medium">{item.delivery_count.toLocaleString()}</span>
                        </div>
                        <div className="flex items-center justify-between">
                          <span className="text-muted-foreground">Errors</span>
                          <span className="font-medium">{item.error_count}</span>
                        </div>
                        <div className="flex items-center justify-between">
                          <span className="text-muted-foreground">Last used</span>
                          <span className="text-right text-xs">
                            {item.last_used_at ? <RelativeTime date={item.last_used_at} /> : "Never"}
                          </span>
                        </div>
                        <div className="flex items-center justify-between">
                          <span className="text-muted-foreground">Updated</span>
                          <span className="text-right text-xs">
                            <RelativeTime date={item.updated_at} />
                          </span>
                        </div>
                      </div>

                      {summary.length > 0 ? (
                        <div className="rounded-lg border bg-muted/30 p-3">
                          <div className="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">
                            Configuration
                          </div>
                          <div className="space-y-1.5 text-sm">
                            {summary.slice(0, 3).map((entry) => (
                              <div key={entry.label} className="flex items-start justify-between gap-3">
                                <span className="text-muted-foreground">{entry.label}</span>
                                <span className="max-w-[65%] break-all text-right">{entry.value}</span>
                              </div>
                            ))}
                          </div>
                        </div>
                      ) : null}

                      {item.error_message ? (
                        <Alert variant={item.status === "error" ? "destructive" : "default"}>
                          {item.status === "error" ? <AlertTriangle className="h-4 w-4" /> : <AlertTriangle className="h-4 w-4" />}
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
                        <Button asChild variant="outline" size="sm">
                          <Link href={`/admin/integrations/${item.id}`}>View Details</Link>
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => {
                            setEditing(item);
                            setDialogType(item.type);
                            setDialogOpen(true);
                          }}
                        >
                          <Settings2 className="mr-2 h-4 w-4" />
                          Edit
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => void handleTest(item.id)}
                          disabled={busyKey === `test:${item.id}`}
                        >
                          <TestTube2 className="mr-2 h-4 w-4" />
                          Test
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => void handleRetry(item.id)}
                          disabled={busyKey === `retry:${item.id}`}
                        >
                          <RotateCcw className="mr-2 h-4 w-4" />
                          Retry Failed
                        </Button>
                        {item.status !== "setup_pending" ? (
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => void handleStatus(item.id, nextStatus)}
                            disabled={busyKey === `status:${item.id}`}
                          >
                            <ToggleLeft className="mr-2 h-4 w-4" />
                            {statusActionLabel}
                          </Button>
                        ) : null}
                        <Button variant="outline" size="sm" onClick={() => setDeleteCandidate(item)}>
                          <Trash2 className="mr-2 h-4 w-4" />
                          Delete
                        </Button>
                      </div>
                    </CardContent>
                  </Card>
                );
              })}
            </div>
          )}
        </section>

        <IntegrationFormDialog
          open={dialogOpen}
          onOpenChange={(next) => {
            setDialogOpen(next);
            if (!next) {
              setEditing(null);
              setDialogType(null);
            }
          }}
          onSaved={handleSaved}
          providers={providers}
          integration={editing}
          initialType={dialogType}
        />

        {deleteCandidate ? (
          <ConfirmDialog
            open={Boolean(deleteCandidate)}
            onOpenChange={(next) => !next && setDeleteCandidate(null)}
            title="Delete integration"
            description={`Delete "${deleteCandidate.name}" and cancel its pending deliveries?`}
            confirmLabel="Delete"
            variant="destructive"
            onConfirm={() => handleDelete(deleteCandidate.id)}
          />
        ) : null}
      </div>
    </PermissionRedirect>
  );
}
