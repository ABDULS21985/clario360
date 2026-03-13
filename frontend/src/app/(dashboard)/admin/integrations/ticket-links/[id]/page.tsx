"use client";

import Link from "next/link";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft, ExternalLink, RefreshCw } from "lucide-react";
import { toast } from "sonner";
import { ErrorState } from "@/components/common/error-state";
import { LoadingSkeleton } from "@/components/common/loading-skeleton";
import { PageHeader } from "@/components/common/page-header";
import { PermissionRedirect } from "@/components/common/permission-redirect";
import { RelativeTime } from "@/components/shared/relative-time";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { apiGet } from "@/lib/api";
import type { ApiResponse } from "@/types/api";
import { fetchTicketLink } from "../../_components/integration-utils";

interface Props {
  params: { id: string };
}

export default function TicketLinkDetailPage({ params }: Props) {
  const { id } = params;
  const router = useRouter();
  const [syncing, setSyncing] = useState(false);
  const query = useQuery({
    queryKey: ["integration-ticket-link", id],
    queryFn: () => fetchTicketLink(id),
  });

  const link = query.data;

  const handleSync = async () => {
    setSyncing(true);
    try {
      await apiGet<ApiResponse<{ status: string }>>(`/api/v1/integrations/ticket-links/${id}/sync`);
      toast.success("Ticket link synchronized");
      await query.refetch();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Unable to synchronize ticket link");
    } finally {
      setSyncing(false);
    }
  };

  if (query.isLoading) {
    return (
      <PermissionRedirect permission="tenant:write">
        <LoadingSkeleton variant="card" count={2} />
      </PermissionRedirect>
    );
  }

  if (query.error || !link) {
    return (
      <PermissionRedirect permission="tenant:write">
        <ErrorState message="Failed to load ticket link" onRetry={() => void query.refetch()} />
      </PermissionRedirect>
    );
  }

  return (
    <PermissionRedirect permission="tenant:write">
      <div className="space-y-6">
        <PageHeader
          title={
            <div className="flex items-center gap-3">
              <button
                onClick={() => router.push(`/admin/integrations/${link.integration_id}`)}
                className="flex h-8 w-8 items-center justify-center rounded-full border bg-background text-muted-foreground shadow-sm transition-colors hover:bg-accent"
              >
                <ArrowLeft className="h-4 w-4" />
              </button>
              <span className="truncate">{link.external_key}</span>
            </div>
          }
          description={`Linked ${link.external_system} ticket for ${link.entity_type} ${link.entity_id}`}
          actions={
            <div className="flex gap-2">
              <Button variant="outline" onClick={() => void query.refetch()} disabled={query.isFetching}>
                <RefreshCw className={`mr-2 h-4 w-4 ${query.isFetching ? "animate-spin" : ""}`} />
                Refresh
              </Button>
              <Button variant="outline" onClick={() => void handleSync()} disabled={syncing}>
                <RefreshCw className={`mr-2 h-4 w-4 ${syncing ? "animate-spin" : ""}`} />
                Force Sync
              </Button>
              <Button asChild variant="outline">
                <a href={link.external_url} target="_blank" rel="noreferrer">
                  <ExternalLink className="mr-2 h-4 w-4" />
                  Open Ticket
                </a>
              </Button>
            </div>
          }
        />

        <div className="grid gap-4 md:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">External Record</CardTitle>
              <CardDescription>Current external system metadata and sync status.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              <KeyValue label="System" value={link.external_system} />
              <KeyValue label="Key" value={link.external_key} />
              <KeyValue label="External ID" value={link.external_id} />
              <KeyValue label="Status" value={link.external_status ?? "Unknown"} />
              <KeyValue label="Priority" value={link.external_priority ?? "Unknown"} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Clario Linkage</CardTitle>
              <CardDescription>How this external ticket is mapped back into Clario 360.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              <KeyValue label="Integration" value={link.integration_id} />
              <KeyValue label="Entity type" value={link.entity_type} />
              <KeyValue label="Entity ID" value={link.entity_id} />
              <KeyValue label="Sync direction" value={link.sync_direction} />
              <KeyValue label="Last sync direction" value={link.last_sync_direction ?? "Unknown"} />
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Timestamps</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-3 md:grid-cols-3 text-sm">
            <div>
              <div className="text-xs uppercase tracking-wide text-muted-foreground">Created</div>
              <div className="mt-1">
                <RelativeTime date={link.created_at} />
              </div>
            </div>
            <div>
              <div className="text-xs uppercase tracking-wide text-muted-foreground">Updated</div>
              <div className="mt-1">
                <RelativeTime date={link.updated_at} />
              </div>
            </div>
            <div>
              <div className="text-xs uppercase tracking-wide text-muted-foreground">Last synced</div>
              <div className="mt-1">{link.last_synced_at ? <RelativeTime date={link.last_synced_at} /> : "Never"}</div>
            </div>
          </CardContent>
        </Card>

        {link.sync_error ? (
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Last Sync Error</CardTitle>
            </CardHeader>
            <CardContent className="text-sm text-destructive">{link.sync_error}</CardContent>
          </Card>
        ) : null}

        <div className="flex gap-2">
          <Button asChild variant="outline">
            <Link href={`/admin/integrations/${link.integration_id}`}>Back to Integration</Link>
          </Button>
        </div>
      </div>
    </PermissionRedirect>
  );
}

function KeyValue({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-xs uppercase tracking-wide text-muted-foreground">{label}</div>
      <div className="mt-1 break-all">{value}</div>
    </div>
  );
}
