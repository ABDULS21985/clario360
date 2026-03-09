"use client";

import { useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import {
  Building2,
  Settings,
  Palette,
  Users,
  ClipboardList,
  ArrowLeft,
  Ban,
  CheckCircle,
  Trash2,
} from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { PageHeader } from "@/components/common/page-header";
import { ErrorState } from "@/components/common/error-state";
import { StatusBadge } from "@/components/shared/status-badge";
import { KpiCard } from "@/components/shared/kpi-card";
import { RelativeTime } from "@/components/shared/relative-time";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { tenantStatusConfig, tenantPlanConfig } from "@/lib/status-configs";
import { formatBytes, formatNumber, formatCompactNumber } from "@/lib/format";
import { useTenant, useTenantUsage, useDeprovisionTenant } from "@/hooks/use-tenants";
import { TenantSettingsForm } from "./tenant-settings-form";
import { TenantBrandingForm } from "./tenant-branding-form";
import api from "@/lib/api";
import Link from "next/link";

interface TenantDetailContentProps {
  tenantId: string;
}

export function TenantDetailContent({ tenantId }: TenantDetailContentProps) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const defaultTab = searchParams?.get("tab") ?? "overview";

  const [deprovisionOpen, setDeprovisionOpen] = useState(false);
  const deprovisionMutation = useDeprovisionTenant();

  const isPollingStatus = (status: string) =>
    status === "provisioning" || status === "deprovisioning";

  const {
    data: tenant,
    isLoading,
    error,
    refetch,
  } = useTenant(tenantId, false);

  // Enable polling only when tenant is in a transitional state
  const { data: polledTenant } = useTenant(
    tenantId,
    tenant ? isPollingStatus(tenant.status) : false,
  );

  const activeTenant = polledTenant ?? tenant;

  const {
    data: usage,
    isLoading: usageLoading,
  } = useTenantUsage(tenantId);

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-10 w-64" />
        <div className="grid grid-cols-4 gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-28" />
          ))}
        </div>
        <Skeleton className="h-96" />
      </div>
    );
  }

  if (error || !activeTenant) {
    return (
      <ErrorState
        message={error?.message ?? "Failed to load tenant details"}
        onRetry={() => refetch()}
      />
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-2">
        <Button variant="ghost" size="icon" asChild>
          <Link href="/admin/tenants" aria-label="Back to tenants">
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <PageHeader
          title={
            <div className="flex items-center gap-3">
              {activeTenant.name}
              <StatusBadge status={activeTenant.status} config={tenantStatusConfig} />
              <StatusBadge status={activeTenant.plan} config={tenantPlanConfig} variant="outline" />
            </div>
          }
          description={
            <span>
              <code className="text-xs font-mono">{activeTenant.slug}</code>
              {activeTenant.domain && (
                <span className="ml-2 text-muted-foreground">· {activeTenant.domain}</span>
              )}
            </span>
          }
          actions={
            <div className="flex items-center gap-2">
              {activeTenant.status === "active" && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={async () => {
                    try {
                      await api.put(`/api/v1/tenants/${tenantId}`, { status: "suspended" });
                      toast.success("Tenant suspended");
                      refetch();
                    } catch {
                      toast.error("Failed to suspend tenant");
                    }
                  }}
                >
                  <Ban className="mr-2 h-4 w-4" />
                  Suspend
                </Button>
              )}
              {activeTenant.status === "suspended" && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={async () => {
                    try {
                      await api.put(`/api/v1/tenants/${tenantId}`, { status: "active" });
                      toast.success("Tenant activated");
                      refetch();
                    } catch {
                      toast.error("Failed to activate tenant");
                    }
                  }}
                >
                  <CheckCircle className="mr-2 h-4 w-4" />
                  Activate
                </Button>
              )}
              {activeTenant.status !== "deprovisioned" && activeTenant.status !== "deprovisioning" && (
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={() => setDeprovisionOpen(true)}
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  Deprovision
                </Button>
              )}
            </div>
          }
        />
      </div>

      {isPollingStatus(activeTenant.status) && (
        <Card className="border-blue-300 bg-blue-50 dark:bg-blue-900/20">
          <CardContent className="p-4 flex items-center gap-3">
            <div className="h-4 w-4 animate-spin rounded-full border-2 border-blue-600 border-t-transparent" />
            <p className="text-sm text-blue-800 dark:text-blue-200">
              Tenant is {activeTenant.status}. Status updates automatically every few seconds.
            </p>
          </CardContent>
        </Card>
      )}

      <Tabs defaultValue={defaultTab}>
        <TabsList>
          <TabsTrigger value="overview" className="gap-2">
            <Building2 className="h-4 w-4" />
            Overview
          </TabsTrigger>
          <TabsTrigger value="settings" className="gap-2">
            <Settings className="h-4 w-4" />
            Settings
          </TabsTrigger>
          <TabsTrigger value="branding" className="gap-2">
            <Palette className="h-4 w-4" />
            Branding
          </TabsTrigger>
          <TabsTrigger value="users" className="gap-2">
            <Users className="h-4 w-4" />
            Users
          </TabsTrigger>
          <TabsTrigger value="audit" className="gap-2">
            <ClipboardList className="h-4 w-4" />
            Audit Log
          </TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <KpiCard
              title="Active Users"
              value={formatNumber(usage?.active_users ?? activeTenant.user_count)}
              icon={Users}
              loading={usageLoading}
            />
            <KpiCard
              title="API Calls"
              value={formatCompactNumber(usage?.api_calls ?? 0)}
              icon={Settings}
              loading={usageLoading}
            />
            <KpiCard
              title="Storage Used"
              value={formatBytes(usage?.storage_used_bytes ?? activeTenant.storage_used_bytes)}
              description={`of ${activeTenant.settings.max_storage_gb} GB`}
              loading={usageLoading}
            />
            <KpiCard
              title="Bandwidth"
              value={formatBytes(usage?.bandwidth_bytes ?? 0)}
              loading={usageLoading}
            />
          </div>

          <Card>
            <CardHeader>
              <CardTitle>Tenant Information</CardTitle>
            </CardHeader>
            <CardContent>
              <dl className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
                <div>
                  <dt className="text-muted-foreground">Tenant ID</dt>
                  <dd className="font-mono text-xs mt-1">{activeTenant.id}</dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">Owner</dt>
                  <dd className="mt-1">{activeTenant.owner_id}</dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">Created</dt>
                  <dd className="mt-1">
                    <RelativeTime date={activeTenant.created_at} />
                  </dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">Provisioned</dt>
                  <dd className="mt-1">
                    {activeTenant.provisioned_at ? (
                      <RelativeTime date={activeTenant.provisioned_at} />
                    ) : (
                      <span className="text-muted-foreground">—</span>
                    )}
                  </dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">MFA Required</dt>
                  <dd className="mt-1">
                    <Badge variant={activeTenant.settings.mfa_required ? "default" : "secondary"}>
                      {activeTenant.settings.mfa_required ? "Yes" : "No"}
                    </Badge>
                  </dd>
                </div>
                <div>
                  <dt className="text-muted-foreground">Session Timeout</dt>
                  <dd className="mt-1">{activeTenant.settings.session_timeout_minutes} minutes</dd>
                </div>
              </dl>
            </CardContent>
          </Card>

          {usage?.suite_usage && Object.keys(usage.suite_usage).length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle>Suite Usage</CardTitle>
                <CardDescription>Usage breakdown by platform suite</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {Object.entries(usage.suite_usage).map(([key, su]) => (
                    <div key={key} className="rounded-lg border p-4">
                      <p className="font-medium text-sm capitalize">{su.suite}</p>
                      <div className="mt-2 space-y-1 text-xs text-muted-foreground">
                        <p>API Calls: {formatCompactNumber(su.api_calls)}</p>
                        <p>Active Users: {su.active_users}</p>
                        {su.last_accessed && (
                          <p>
                            Last accessed: <RelativeTime date={su.last_accessed} />
                          </p>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="settings">
          <TenantSettingsForm tenant={activeTenant} onSuccess={() => refetch()} />
        </TabsContent>

        <TabsContent value="branding">
          <TenantBrandingForm tenant={activeTenant} onSuccess={() => refetch()} />
        </TabsContent>

        <TabsContent value="users">
          <Card>
            <CardHeader>
              <CardTitle>Tenant Users</CardTitle>
              <CardDescription>Users belonging to this tenant</CardDescription>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">
                View and manage users for this tenant in the{" "}
                <Link href={`/admin/users?tenant_id=${tenantId}`} className="text-primary hover:underline">
                  Users section
                </Link>.
              </p>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="audit">
          <Card>
            <CardHeader>
              <CardTitle>Audit Log</CardTitle>
              <CardDescription>Activity log for this tenant</CardDescription>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">
                View audit logs for this tenant in the{" "}
                <Link href={`/admin/audit?tenant_id=${tenantId}`} className="text-primary hover:underline">
                  Audit Logs section
                </Link>.
              </p>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      <ConfirmDialog
        open={deprovisionOpen}
        onOpenChange={setDeprovisionOpen}
        title="Deprovision Tenant"
        description={`This will permanently deprovision "${activeTenant.name}" and all associated data. This action cannot be undone.`}
        confirmLabel="Deprovision"
        typeToConfirm={activeTenant.name}
        variant="destructive"
        loading={deprovisionMutation.isPending}
        onConfirm={async () => {
          await deprovisionMutation.mutateAsync(tenantId);
          refetch();
        }}
      />
    </div>
  );
}
