'use client';

import { useState } from 'react';
import Link from 'next/link';
import { ArrowRight, Building2, Palette, Settings2 } from 'lucide-react';
import { PageHeader } from '@/components/common/page-header';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { KpiCard } from '@/components/shared/kpi-card';
import { RelativeTime } from '@/components/shared/relative-time';
import { StatusBadge } from '@/components/shared/status-badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useAuth } from '@/hooks/use-auth';
import { useTenant, useTenants, useTenantUsage } from '@/hooks/use-tenants';
import { tenantPlanConfig, tenantStatusConfig } from '@/lib/status-configs';
import { TenantBrandingForm } from '../../tenants/[tenantId]/_components/tenant-branding-form';
import { TenantSettingsForm } from '../../tenants/[tenantId]/_components/tenant-settings-form';

function formatStorage(bytes: number): string {
  const gb = bytes / (1024 * 1024 * 1024);
  if (gb >= 1) {
    return `${gb.toFixed(gb >= 10 ? 0 : 1)} GB`;
  }

  const mb = bytes / (1024 * 1024);
  return `${Math.round(mb)} MB`;
}

export function CurrentTenantSettings() {
  const { user, hasPermission } = useAuth();
  const isSuperAdmin = hasPermission('*');
  const ownTenantId = user?.tenant_id ?? null;
  const [selectedTenantId, setSelectedTenantId] = useState<string | null>(null);
  const tenantId = selectedTenantId ?? ownTenantId;

  // Fetch tenant list for super-admin picker
  const { data: tenantsPage } = useTenants(
    isSuperAdmin ? { page: 1, per_page: 100 } : undefined,
  );
  const tenantOptions = tenantsPage?.data ?? [];

  const {
    data: tenant,
    isLoading: tenantLoading,
    error: tenantError,
    refetch: refetchTenant,
  } = useTenant(tenantId ?? '', false);
  const {
    data: usage,
    isLoading: usageLoading,
    refetch: refetchUsage,
  } = useTenantUsage(tenantId ?? '');

  const enabledSuites = tenant?.settings?.enabled_suites ?? [];

  if (!tenantId) {
    return (
      <PermissionRedirect permission="tenant:write">
        <ErrorState
          title="Tenant context unavailable"
          message="Current tenant information is not available for this session."
        />
      </PermissionRedirect>
    );
  }

  if (tenantLoading) {
    return (
      <PermissionRedirect permission="tenant:write">
        <div className="space-y-6">
          <LoadingSkeleton variant="card" count={1} />
          <LoadingSkeleton variant="card" count={3} />
        </div>
      </PermissionRedirect>
    );
  }

  if (tenantError || !tenant) {
    return (
      <PermissionRedirect permission="tenant:write">
        <ErrorState
          title="Unable to load tenant settings"
          message={tenantError?.message ?? 'The tenant configuration could not be loaded.'}
          onRetry={() => {
            void refetchTenant();
            void refetchUsage();
          }}
        />
      </PermissionRedirect>
    );
  }

  return (
    <PermissionRedirect permission="tenant:write">
      <div className="space-y-6">
        <PageHeader
          title="Platform Settings"
          description="Manage the active tenant configuration, branding, and platform limits using the live tenant contract."
          actions={
            <div className="flex items-center gap-3">
              {isSuperAdmin && tenantOptions.length > 1 && (
                <Select
                  value={tenantId ?? ''}
                  onValueChange={(v) => setSelectedTenantId(v === ownTenantId ? null : v)}
                >
                  <SelectTrigger className="w-[220px] h-9 text-sm">
                    <SelectValue placeholder="Select tenant" />
                  </SelectTrigger>
                  <SelectContent>
                    {tenantOptions.map((t) => (
                      <SelectItem key={t.id} value={t.id}>
                        {t.name}{t.id === ownTenantId ? ' (current)' : ''}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
              <Button asChild variant="outline" size="sm">
                <Link href={`/admin/tenants/${tenant.id}`}>
                  Open tenant record
                  <ArrowRight className="ml-1.5 h-4 w-4" />
                </Link>
              </Button>
            </div>
          }
        />

        <div className="grid grid-cols-1 gap-4 xl:grid-cols-4">
          <KpiCard
            title="Tenant Status"
            value={tenant.status}
            description={tenant.domain ?? tenant.slug}
            icon={Building2}
            loading={tenantLoading}
          />
          <KpiCard
            title="Subscription"
            value={tenant.subscription_tier}
            description={`${enabledSuites.length} suite${enabledSuites.length === 1 ? '' : 's'} enabled`}
            icon={Settings2}
            loading={tenantLoading}
          />
          <KpiCard
            title="Storage Used"
            value={usage ? formatStorage(usage.storage_used_bytes ?? 0) : 'Unavailable'}
            description={
              tenant.settings?.max_storage_gb
                ? `Limit ${tenant.settings.max_storage_gb} GB`
                : 'No storage limit configured'
            }
            icon={Palette}
            loading={usageLoading}
          />
          <KpiCard
            title="Active Users"
            value={usage?.active_users ?? 0}
            description={tenant.settings?.max_users ? `Limit ${tenant.settings.max_users}` : 'No user limit configured'}
            icon={Building2}
            loading={usageLoading}
          />
        </div>

        <Card>
          <CardHeader>
            <CardTitle className="flex flex-wrap items-center gap-3">
              {tenant.name}
              <StatusBadge status={tenant.status} config={tenantStatusConfig} />
              <StatusBadge status={tenant.subscription_tier} config={tenantPlanConfig} variant="outline" />
            </CardTitle>
            <CardDescription>Tenant-wide settings stored in the IAM tenant record.</CardDescription>
          </CardHeader>
          <CardContent className="grid grid-cols-1 gap-4 text-sm md:grid-cols-2 xl:grid-cols-4">
            <div>
              <p className="text-muted-foreground">Tenant ID</p>
              <p className="mt-1 font-mono text-xs">{tenant.id}</p>
            </div>
            <div>
              <p className="text-muted-foreground">Slug</p>
              <p className="mt-1 font-medium">{tenant.slug}</p>
            </div>
            <div>
              <p className="text-muted-foreground">Created</p>
              <p className="mt-1 font-medium">
                <RelativeTime date={tenant.created_at} />
              </p>
            </div>
            <div>
              <p className="text-muted-foreground">Updated</p>
              <p className="mt-1 font-medium">
                <RelativeTime date={tenant.updated_at} />
              </p>
            </div>
          </CardContent>
        </Card>

        <Tabs defaultValue="settings" className="space-y-6">
          <TabsList>
            <TabsTrigger value="settings">Configuration</TabsTrigger>
            <TabsTrigger value="branding">Branding</TabsTrigger>
            <TabsTrigger value="usage">Usage</TabsTrigger>
          </TabsList>

          <TabsContent value="settings">
            <TenantSettingsForm
              tenant={tenant}
              onSuccess={() => {
                void refetchTenant();
                void refetchUsage();
              }}
            />
          </TabsContent>

          <TabsContent value="branding">
            <TenantBrandingForm
              tenant={tenant}
              onSuccess={() => {
                void refetchTenant();
              }}
            />
          </TabsContent>

          <TabsContent value="usage">
            <Card>
              <CardHeader>
                <CardTitle>Tenant Usage</CardTitle>
                <CardDescription>Current consumption and enabled suite footprint.</CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
                  <KpiCard title="API Calls" value={usage?.api_calls ?? 0} loading={usageLoading} />
                  <KpiCard title="Bandwidth" value={usage ? formatStorage(usage.bandwidth_bytes ?? 0) : 'Unavailable'} loading={usageLoading} />
                  <KpiCard title="Enabled Suites" value={enabledSuites.length} description={enabledSuites.join(', ') || 'None'} />
                </div>

                <div className="space-y-3">
                  <h3 className="text-sm font-semibold">Suite Usage</h3>
                  {usage && Object.keys(usage.suite_usage ?? {}).length > 0 ? (
                    <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
                      {Object.entries(usage.suite_usage).map(([suite, item]) => (
                        <div key={suite} className="rounded-xl border p-4">
                          <div className="flex items-center justify-between gap-3">
                            <p className="font-medium capitalize">{suite}</p>
                            <span className="text-xs text-muted-foreground">
                              {item.last_accessed ? <RelativeTime date={item.last_accessed} /> : 'No recent activity'}
                            </span>
                          </div>
                          <div className="mt-3 grid grid-cols-2 gap-3 text-sm">
                            <div>
                              <p className="text-muted-foreground">API Calls</p>
                              <p className="font-semibold">{item.api_calls}</p>
                            </div>
                            <div>
                              <p className="text-muted-foreground">Active Users</p>
                              <p className="font-semibold">{item.active_users}</p>
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="text-sm text-muted-foreground">Usage telemetry is not currently available for this tenant.</p>
                  )}
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </PermissionRedirect>
  );
}
