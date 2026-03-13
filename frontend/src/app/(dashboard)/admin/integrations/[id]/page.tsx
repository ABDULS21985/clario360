"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowLeft,
  RefreshCw,
  RotateCcw,
  Settings2,
  TestTube2,
  ToggleLeft,
  Trash2,
} from "lucide-react";
import { toast } from "sonner";
import { ErrorState } from "@/components/common/error-state";
import { LoadingSkeleton } from "@/components/common/loading-skeleton";
import { PageHeader } from "@/components/common/page-header";
import { PermissionRedirect } from "@/components/common/permission-redirect";
import { RelativeTime } from "@/components/shared/relative-time";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { apiDelete, apiGet, apiPost, apiPut } from "@/lib/api";
import type { ApiResponse } from "@/types/api";
import type { IntegrationStatus } from "@/types/integration";
import { DeliveryLogTable } from "../_components/delivery-log-table";
import { IntegrationFormDialog } from "../_components/integration-form-dialog";
import { TicketLinksTable } from "../_components/ticket-links-table";
import {
  fetchDeliveries,
  fetchIntegration,
  fetchProviders,
  fetchTicketLinks,
  getSetupPendingFields,
  prettyType,
  statusBadgeVariant,
  summarizeIntegrationConfig,
} from "../_components/integration-utils";

interface Props {
  params: { id: string };
}

export default function IntegrationDetailPage({ params }: Props) {
  const { id } = params;
  const router = useRouter();
  const [editOpen, setEditOpen] = useState(false);
  const [busyKey, setBusyKey] = useState<string | null>(null);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deliveryStatus, setDeliveryStatus] = useState("all");
  const [deliveryEventType, setDeliveryEventType] = useState("");
  const [deliveryPage, setDeliveryPage] = useState(1);
  const [syncingTicketId, setSyncingTicketId] = useState<string | null>(null);

  const providersQuery = useQuery({
    queryKey: ["integration-providers"],
    queryFn: fetchProviders,
  });
  const integrationQuery = useQuery({
    queryKey: ["integration-detail", id],
    queryFn: () => fetchIntegration(id),
    refetchInterval: 30000,
  });
  const deliveriesQuery = useQuery({
    queryKey: ["integration-deliveries", id, deliveryPage, deliveryStatus, deliveryEventType],
    queryFn: () =>
      fetchDeliveries(id, {
        page: deliveryPage,
        per_page: 20,
        status: deliveryStatus === "all" ? undefined : deliveryStatus,
        event_type: deliveryEventType.trim() || undefined,
      }),
  });
  const ticketLinksQuery = useQuery({
    queryKey: ["integration-ticket-links", id],
    queryFn: () => fetchTicketLinks(id),
  });

  const integration = integrationQuery.data;
  const providers = providersQuery.data ?? [];
  const setupPendingFields = integration ? getSetupPendingFields(integration) : [];
  const configSummary = useMemo(() => (integration ? summarizeIntegrationConfig(integration) : []), [integration]);

  const handleTest = async () => {
    setBusyKey("test");
    try {
      const response = await apiPost<ApiResponse<{ response_code: number }>>(`/api/v1/integrations/${id}/test`);
      toast.success(`Test completed with HTTP ${response.data.response_code}`);
      await deliveriesQuery.refetch();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Integration test failed");
    } finally {
      setBusyKey(null);
    }
  };

  const handleRetry = async () => {
    setBusyKey("retry");
    try {
      const response = await apiPost<ApiResponse<{ retried_count: number }>>(`/api/v1/integrations/${id}/retry-failed`);
      toast.success(`Re-queued ${response.data.retried_count} failed deliveries`);
      await deliveriesQuery.refetch();
      await integrationQuery.refetch();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Retry failed");
    } finally {
      setBusyKey(null);
    }
  };

  const handleStatus = async (nextStatus: Exclude<IntegrationStatus, "setup_pending">) => {
    setBusyKey("status");
    try {
      await apiPut<ApiResponse<{ status: string }>>(`/api/v1/integrations/${id}/status`, { status: nextStatus });
      toast.success(`Integration marked ${nextStatus}`);
      await integrationQuery.refetch();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Status update failed");
    } finally {
      setBusyKey(null);
    }
  };

  const handleDelete = async () => {
    setBusyKey("delete");
    try {
      await apiDelete(`/api/v1/integrations/${id}`);
      toast.success("Integration deleted");
      router.push("/admin/integrations");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Delete failed");
      throw err;
    } finally {
      setBusyKey(null);
    }
  };

  const handleSyncTicketLink = async (linkID: string) => {
    setSyncingTicketId(linkID);
    try {
      await apiGetSync(linkID);
      toast.success("Ticket link synchronized");
      await ticketLinksQuery.refetch();
      await deliveriesQuery.refetch();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Ticket link sync failed");
    } finally {
      setSyncingTicketId(null);
    }
  };

  if (integrationQuery.isLoading) {
    return (
      <PermissionRedirect permission="tenant:write">
        <div className="space-y-6">
          <LoadingSkeleton variant="card" count={2} />
        </div>
      </PermissionRedirect>
    );
  }

  if (integrationQuery.error || !integration) {
    return (
      <PermissionRedirect permission="tenant:write">
        <ErrorState
          title="Integration unavailable"
          message="The selected integration could not be loaded."
          onRetry={() => void integrationQuery.refetch()}
        />
      </PermissionRedirect>
    );
  }

  const nextStatus = integration.status === "active" ? "inactive" : "active";
  const summaryDetails =
    configSummary.length > 0
      ? configSummary
      : [{ label: "Configuration", value: "No non-secret configuration values available." }];

  return (
    <PermissionRedirect permission="tenant:write">
      <div className="space-y-6">
        <PageHeader
          title={
            <div className="flex items-center gap-3">
              <button
                onClick={() => router.push("/admin/integrations")}
                className="flex h-8 w-8 items-center justify-center rounded-full border bg-background text-muted-foreground shadow-sm transition-colors hover:bg-accent"
              >
                <ArrowLeft className="h-4 w-4" />
              </button>
              <span className="truncate">{integration.name}</span>
            </div>
          }
          description={
            <div className="flex flex-wrap items-center gap-3 pl-11 text-sm">
              <Badge variant={statusBadgeVariant(integration.status)}>{integration.status.replace("_", " ")}</Badge>
              <span className="text-muted-foreground">{prettyType(integration.type)}</span>
              <span className="text-muted-foreground">Updated <RelativeTime date={integration.updated_at} /></span>
            </div>
          }
          actions={
            <div className="flex flex-wrap gap-2">
              <Button variant="outline" onClick={() => void integrationQuery.refetch()} disabled={integrationQuery.isFetching}>
                <RefreshCw className={`mr-2 h-4 w-4 ${integrationQuery.isFetching ? "animate-spin" : ""}`} />
                Refresh
              </Button>
              <Button variant="outline" onClick={() => setEditOpen(true)}>
                <Settings2 className="mr-2 h-4 w-4" />
                Edit
              </Button>
              <Button variant="outline" onClick={() => void handleTest()} disabled={busyKey === "test"}>
                <TestTube2 className="mr-2 h-4 w-4" />
                Test
              </Button>
              <Button variant="outline" onClick={() => void handleRetry()} disabled={busyKey === "retry"}>
                <RotateCcw className="mr-2 h-4 w-4" />
                Retry Failed
              </Button>
              {integration.status !== "setup_pending" ? (
                <Button variant="outline" onClick={() => void handleStatus(nextStatus)} disabled={busyKey === "status"}>
                  <ToggleLeft className="mr-2 h-4 w-4" />
                  {integration.status === "active" ? "Disable" : "Enable"}
                </Button>
              ) : null}
              <Button variant="outline" onClick={() => setDeleteOpen(true)}>
                <Trash2 className="mr-2 h-4 w-4" />
                Delete
              </Button>
            </div>
          }
        />

        {integration.status === "setup_pending" ? (
          <Alert>
            <AlertTitle>Setup still needs completion</AlertTitle>
            <AlertDescription>
              {setupPendingFields.length > 0
                ? `Complete the missing fields to move this integration to active: ${setupPendingFields.join(", ")}.`
                : "This integration is still pending completion. Review the configuration below and save the required fields."}
            </AlertDescription>
          </Alert>
        ) : null}

        <div className="grid gap-4 md:grid-cols-3">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Usage</CardTitle>
              <CardDescription>Delivery and runtime health for this integration.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              <KeyValue label="Deliveries" value={integration.delivery_count.toLocaleString()} />
              <KeyValue label="Consecutive errors" value={String(integration.error_count)} />
              <KeyValue label="Last used" value={integration.last_used_at ? "recently" : "never"} />
              <KeyValue label="Created" value={new Date(integration.created_at).toLocaleString()} />
            </CardContent>
          </Card>
          <Card className="md:col-span-2">
            <CardHeader>
              <CardTitle className="text-base">Configuration Summary</CardTitle>
              <CardDescription>Sanitized, non-secret values currently stored for this integration.</CardDescription>
            </CardHeader>
            <CardContent className="grid gap-3 md:grid-cols-2">
              {summaryDetails.map((entry) => (
                <KeyValue key={entry.label} label={entry.label} value={entry.value} />
              ))}
            </CardContent>
          </Card>
        </div>

        {integration.error_message ? (
          <Alert variant={integration.status === "error" ? "destructive" : "default"}>
            <AlertTitle>{integration.status === "error" ? "Integration error" : "Attention required"}</AlertTitle>
            <AlertDescription>{integration.error_message}</AlertDescription>
          </Alert>
        ) : null}

        <Tabs defaultValue="overview">
          <TabsList className="w-full justify-start overflow-x-auto">
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="deliveries">Delivery Log</TabsTrigger>
            <TabsTrigger value="tickets">Ticket Links</TabsTrigger>
          </TabsList>

          <TabsContent value="overview" className="mt-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Integration Notes</CardTitle>
                <CardDescription>{integration.description || "No description recorded."}</CardDescription>
              </CardHeader>
              <CardContent className="grid gap-3 md:grid-cols-2">
                <KeyValue label="Type" value={prettyType(integration.type)} />
                <KeyValue label="Status" value={integration.status} />
                <KeyValue label="Updated" value={new Date(integration.updated_at).toLocaleString()} />
                <KeyValue label="Created by" value={integration.created_by} />
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="deliveries" className="mt-4 space-y-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Filters</CardTitle>
                <CardDescription>Slice the delivery log by status or event type.</CardDescription>
              </CardHeader>
              <CardContent className="grid gap-4 md:grid-cols-3">
                <div className="space-y-2">
                  <div className="text-sm font-medium">Status</div>
                  <Select
                    value={deliveryStatus}
                    onValueChange={(value) => {
                      setDeliveryPage(1);
                      setDeliveryStatus(value);
                    }}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">All</SelectItem>
                      <SelectItem value="pending">Pending</SelectItem>
                      <SelectItem value="retrying">Retrying</SelectItem>
                      <SelectItem value="delivered">Delivered</SelectItem>
                      <SelectItem value="failed">Failed</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2 md:col-span-2">
                  <div className="text-sm font-medium">Event type</div>
                  <Input
                    value={deliveryEventType}
                    onChange={(event) => setDeliveryEventType(event.target.value)}
                    onBlur={() => setDeliveryPage(1)}
                    placeholder="alert.created"
                  />
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="text-base">Delivery Records</CardTitle>
                <CardDescription>
                  {deliveriesQuery.data?.meta.total ?? 0} matching record(s) across this integration.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                {deliveriesQuery.isLoading ? (
                  <LoadingSkeleton variant="table-row" count={6} />
                ) : deliveriesQuery.error ? (
                  <ErrorState message="Failed to load delivery records" onRetry={() => void deliveriesQuery.refetch()} />
                ) : (
                  <>
                    <DeliveryLogTable items={deliveriesQuery.data?.data ?? []} />
                    <PaginationControls
                      page={deliveriesQuery.data?.meta.page ?? 1}
                      totalPages={deliveriesQuery.data?.meta.total_pages ?? 1}
                      onPrev={() => setDeliveryPage((current) => Math.max(1, current - 1))}
                      onNext={() =>
                        setDeliveryPage((current) =>
                          Math.min(deliveriesQuery.data?.meta.total_pages ?? current, current + 1),
                        )
                      }
                    />
                  </>
                )}
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="tickets" className="mt-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">External Ticket Links</CardTitle>
                <CardDescription>Bidirectional sync state for Jira or ServiceNow tickets linked through this integration.</CardDescription>
              </CardHeader>
              <CardContent>
                {ticketLinksQuery.isLoading ? (
                  <LoadingSkeleton variant="table-row" count={4} />
                ) : ticketLinksQuery.error ? (
                  <ErrorState message="Failed to load ticket links" onRetry={() => void ticketLinksQuery.refetch()} />
                ) : (
                  <TicketLinksTable
                    items={ticketLinksQuery.data ?? []}
                    syncingId={syncingTicketId}
                    onSync={(linkID) => void handleSyncTicketLink(linkID)}
                  />
                )}
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>

        <IntegrationFormDialog
          open={editOpen}
          onOpenChange={setEditOpen}
          onSaved={(updated) => {
            toast.success("Integration configuration updated");
            setEditOpen(false);
            integrationQuery.refetch();
          }}
          providers={providers}
          integration={integration}
          initialType={integration.type}
        />

        <ConfirmDialog
          open={deleteOpen}
          onOpenChange={setDeleteOpen}
          title="Delete integration"
          description={`Delete "${integration.name}" and cancel all pending or retrying deliveries?`}
          confirmLabel="Delete"
          variant="destructive"
          onConfirm={handleDelete}
        />
      </div>
    </PermissionRedirect>
  );
}

function KeyValue({ label, value }: { label: string; value: string }) {
  return (
    <div className="space-y-1">
      <div className="text-xs uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className="break-all text-sm">{value}</div>
    </div>
  );
}

function PaginationControls({
  page,
  totalPages,
  onPrev,
  onNext,
}: {
  page: number;
  totalPages: number;
  onPrev: () => void;
  onNext: () => void;
}) {
  return (
    <div className="flex items-center justify-between">
      <div className="text-sm text-muted-foreground">
        Page {page} of {totalPages}
      </div>
      <div className="flex gap-2">
        <Button variant="outline" size="sm" onClick={onPrev} disabled={page <= 1}>
          Previous
        </Button>
        <Button variant="outline" size="sm" onClick={onNext} disabled={page >= totalPages}>
          Next
        </Button>
      </div>
    </div>
  );
}

async function apiGetSync(linkID: string) {
  await apiGet<ApiResponse<{ status: string }>>(`/api/v1/integrations/ticket-links/${linkID}/sync`);
}
