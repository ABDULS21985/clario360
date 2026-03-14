'use client';

import { useState, useMemo, useCallback } from 'react';
import {
  Cloud,
  Database,
  Key,
  Monitor,
  Plus,
  RefreshCw,
  Search,
  Shield,
  TicketCheck,
  Unplug,
  type LucideIcon,
} from 'lucide-react';

import { PageHeader } from '@/components/common/page-header';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { KpiCard } from '@/components/shared/kpi-card';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

import { useRealtimeData } from '@/hooks/use-realtime-data';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { titleCase } from '@/lib/format';
import { cn } from '@/lib/utils';
import type { PaginatedResponse } from '@/types/api';
import type {
  VCISOIntegration,
  IntegrationType,
  IntegrationStatus,
  IntegrationHealth,
} from '@/types/cyber';

import { IntegrationCard } from './_components/integration-card';
import { IntegrationFormDialog } from './_components/integration-form-dialog';
import { IntegrationDetailPanel } from './_components/integration-detail-panel';
import { useSyncIntegration } from './_components/integration-sync-action';

// ─── Category Definitions ───────────────────────────────────────────────────

interface CategoryMeta {
  label: string;
  icon: LucideIcon;
  iconColor: string;
  description: string;
}

const CATEGORY_META: Record<IntegrationType, CategoryMeta> = {
  asset_management: {
    label: 'Asset Management',
    icon: Database,
    iconColor: 'text-purple-600',
    description: 'Asset inventory and CMDB integrations',
  },
  ticketing: {
    label: 'Ticketing',
    icon: TicketCheck,
    iconColor: 'text-orange-600',
    description: 'Issue tracking and service desk',
  },
  cloud_security: {
    label: 'Cloud Security',
    icon: Cloud,
    iconColor: 'text-sky-600',
    description: 'Cloud security posture management',
  },
  data_protection: {
    label: 'Data Protection',
    icon: Shield,
    iconColor: 'text-teal-600',
    description: 'DLP and data classification',
  },
  siem: {
    label: 'SIEM',
    icon: Monitor,
    iconColor: 'text-indigo-600',
    description: 'Security information and event management',
  },
  iam: {
    label: 'IAM',
    icon: Key,
    iconColor: 'text-amber-600',
    description: 'Identity and access management',
  },
};

const ALL_TYPES: IntegrationType[] = [
  'asset_management',
  'ticketing',
  'cloud_security',
  'data_protection',
  'siem',
  'iam',
];

const STATUS_OPTIONS: { label: string; value: IntegrationStatus }[] = [
  { label: 'Connected', value: 'connected' },
  { label: 'Disconnected', value: 'disconnected' },
  { label: 'Error', value: 'error' },
  { label: 'Pending', value: 'pending' },
];

const HEALTH_OPTIONS: { label: string; value: IntegrationHealth }[] = [
  { label: 'Healthy', value: 'healthy' },
  { label: 'Degraded', value: 'degraded' },
  { label: 'Unavailable', value: 'unavailable' },
];

// ─── Main Page Component ────────────────────────────────────────────────────

export default function IntegrationsPage() {
  const [selectedIntegration, setSelectedIntegration] = useState<VCISOIntegration | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [formOpen, setFormOpen] = useState(false);
  const [editIntegration, setEditIntegration] = useState<VCISOIntegration | null>(null);
  const [disconnectTarget, setDisconnectTarget] = useState<VCISOIntegration | null>(null);
  const [disconnectOpen, setDisconnectOpen] = useState(false);

  // ── Filters ─────────────────────────────────────────────────────────────
  const [searchQuery, setSearchQuery] = useState('');
  const [filterType, setFilterType] = useState<string>('all');
  const [filterStatus, setFilterStatus] = useState<string>('all');
  const [filterHealth, setFilterHealth] = useState<string>('all');

  // ── Data Fetch ──────────────────────────────────────────────────────────
  const {
    data: integrationsEnvelope,
    isLoading,
    error,
    mutate: refetch,
  } = useRealtimeData<PaginatedResponse<VCISOIntegration>>(
    API_ENDPOINTS.CYBER_VCISO_INTEGRATIONS,
    {
      params: { per_page: 100 },
      wsTopics: [
        'integration.created',
        'integration.updated',
        'integration.deleted',
        'integration.synced',
      ],
    },
  );

  const integrations = integrationsEnvelope?.data ?? [];

  // ── Sync mutation ───────────────────────────────────────────────────────
  const { triggerSync, syncing } = useSyncIntegration(() => void refetch());

  // ── Disconnect mutation ─────────────────────────────────────────────────
  const { mutate: disconnectIntegration } = useApiMutation<unknown, { id: string }>(
    'delete',
    (variables) => `${API_ENDPOINTS.CYBER_VCISO_INTEGRATIONS}/${variables.id}`,
    {
      successMessage: 'Integration disconnected',
      invalidateKeys: [API_ENDPOINTS.CYBER_VCISO_INTEGRATIONS],
      onSuccess: () => void refetch(),
    },
  );

  // ── Filtered data ───────────────────────────────────────────────────────
  const filteredIntegrations = useMemo(() => {
    let result = integrations;

    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      result = result.filter(
        (i) =>
          i.name.toLowerCase().includes(q) ||
          i.provider.toLowerCase().includes(q),
      );
    }

    if (filterType !== 'all') {
      result = result.filter((i) => i.type === filterType);
    }
    if (filterStatus !== 'all') {
      result = result.filter((i) => i.status === filterStatus);
    }
    if (filterHealth !== 'all') {
      result = result.filter((i) => i.health_status === filterHealth);
    }

    return result;
  }, [integrations, searchQuery, filterType, filterStatus, filterHealth]);

  // ── Category summary stats ─────────────────────────────────────────────
  const categorySummary = useMemo(() => {
    const summary: Record<IntegrationType, { total: number; connected: number; healthy: number }> =
      {} as Record<IntegrationType, { total: number; connected: number; healthy: number }>;

    for (const type of ALL_TYPES) {
      summary[type] = { total: 0, connected: 0, healthy: 0 };
    }

    for (const integration of integrations) {
      const cat = summary[integration.type];
      if (cat) {
        cat.total++;
        if (integration.status === 'connected') cat.connected++;
        if (integration.health_status === 'healthy') cat.healthy++;
      }
    }

    return summary;
  }, [integrations]);

  // ── Global counts ──────────────────────────────────────────────────────
  const totalConnected = integrations.filter((i) => i.status === 'connected').length;
  const totalErrors = integrations.filter((i) => i.status === 'error').length;
  const totalItemsSynced = integrations.reduce((sum, i) => sum + i.items_synced, 0);

  // ── Handlers ────────────────────────────────────────────────────────────
  const handleViewDetails = useCallback((integration: VCISOIntegration) => {
    setSelectedIntegration(integration);
    setDetailOpen(true);
  }, []);

  const handleConfigure = useCallback((integration: VCISOIntegration) => {
    setEditIntegration(integration);
    setFormOpen(true);
  }, []);

  const handleAddNew = useCallback(() => {
    setEditIntegration(null);
    setFormOpen(true);
  }, []);

  const handleDisconnect = useCallback((integration: VCISOIntegration) => {
    setDisconnectTarget(integration);
    setDisconnectOpen(true);
  }, []);

  const confirmDisconnect = useCallback(async () => {
    if (disconnectTarget) {
      disconnectIntegration({ id: disconnectTarget.id });
      setDisconnectTarget(null);
    }
  }, [disconnectTarget, disconnectIntegration]);

  const hasActiveFilters =
    searchQuery.trim() !== '' ||
    filterType !== 'all' ||
    filterStatus !== 'all' ||
    filterHealth !== 'all';

  const clearFilters = useCallback(() => {
    setSearchQuery('');
    setFilterType('all');
    setFilterStatus('all');
    setFilterHealth('all');
  }, []);

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Integrations"
          description="Manage connections to external security tools including asset management, ticketing, cloud security, and data protection platforms."
          actions={
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => void refetch()}
              >
                <RefreshCw className="mr-1.5 h-4 w-4" />
                Refresh
              </Button>
              <Button size="sm" onClick={handleAddNew}>
                <Plus className="mr-1.5 h-4 w-4" />
                Add Integration
              </Button>
            </div>
          }
        />

        {/* KPI Summary Row */}
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <KpiCard
            title="Total Integrations"
            value={integrations.length}
            icon={Database}
            iconColor="text-blue-600"
            loading={isLoading}
            description={`${totalConnected} connected`}
          />
          <KpiCard
            title="Connected"
            value={totalConnected}
            icon={Monitor}
            iconColor="text-green-600"
            loading={isLoading}
            description={`of ${integrations.length} total`}
          />
          <KpiCard
            title="Errors"
            value={totalErrors}
            icon={Shield}
            iconColor="text-red-600"
            loading={isLoading}
            description={totalErrors > 0 ? 'Require attention' : 'All systems nominal'}
            className={totalErrors > 0 ? 'border-red-200' : ''}
          />
          <KpiCard
            title="Total Items Synced"
            value={totalItemsSynced.toLocaleString()}
            icon={RefreshCw}
            iconColor="text-purple-600"
            loading={isLoading}
            description="Across all integrations"
          />
        </div>

        {/* Category Summary Cards */}
        <div className="grid gap-3 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-6">
          {ALL_TYPES.map((type) => {
            const meta = CATEGORY_META[type];
            const stats = categorySummary[type];
            const CategoryIcon = meta.icon;
            return (
              <button
                key={type}
                className={cn(
                  'rounded-xl border bg-card p-3 text-left transition-all hover:shadow-sm hover:border-primary/30',
                  filterType === type && 'border-primary ring-1 ring-primary/20',
                )}
                onClick={() => setFilterType(filterType === type ? 'all' : type)}
              >
                <div className="flex items-center gap-2 mb-2">
                  <CategoryIcon className={cn('h-4 w-4 shrink-0', meta.iconColor)} />
                  <span className="text-xs font-semibold truncate">{meta.label}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-lg font-bold tabular-nums">{stats.total}</span>
                  {stats.connected > 0 && (
                    <Badge
                      variant="secondary"
                      className="text-[10px] bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
                    >
                      {stats.connected} up
                    </Badge>
                  )}
                </div>
              </button>
            );
          })}
        </div>

        {/* Filters Row */}
        <div className="flex flex-wrap items-center gap-3">
          <div className="relative flex-1 min-w-[200px] max-w-xs">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Search integrations..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-9"
            />
          </div>

          <Select value={filterType} onValueChange={setFilterType}>
            <SelectTrigger className="w-[160px]">
              <SelectValue placeholder="All Types" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Types</SelectItem>
              {ALL_TYPES.map((type) => (
                <SelectItem key={type} value={type}>
                  {CATEGORY_META[type].label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Select value={filterStatus} onValueChange={setFilterStatus}>
            <SelectTrigger className="w-[150px]">
              <SelectValue placeholder="All Status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Status</SelectItem>
              {STATUS_OPTIONS.map((opt) => (
                <SelectItem key={opt.value} value={opt.value}>
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Select value={filterHealth} onValueChange={setFilterHealth}>
            <SelectTrigger className="w-[150px]">
              <SelectValue placeholder="All Health" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Health</SelectItem>
              {HEALTH_OPTIONS.map((opt) => (
                <SelectItem key={opt.value} value={opt.value}>
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          {hasActiveFilters && (
            <Button variant="ghost" size="sm" onClick={clearFilters}>
              Clear filters
            </Button>
          )}
        </div>

        {/* Integration Cards Grid */}
        {isLoading ? (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <LoadingSkeleton variant="card" count={6} />
          </div>
        ) : error ? (
          <ErrorState
            message="Failed to load integrations."
            onRetry={() => void refetch()}
          />
        ) : filteredIntegrations.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-center">
            <div className="rounded-full bg-muted p-4 mb-4">
              <Unplug className="h-8 w-8 text-muted-foreground" />
            </div>
            <h3 className="text-base font-medium mb-1">
              {hasActiveFilters ? 'No matching integrations' : 'No integrations yet'}
            </h3>
            <p className="text-sm text-muted-foreground mb-4 max-w-sm">
              {hasActiveFilters
                ? 'Try adjusting your filters to see more results.'
                : 'Connect your first external security tool to start syncing data.'}
            </p>
            {hasActiveFilters ? (
              <Button variant="outline" size="sm" onClick={clearFilters}>
                Clear filters
              </Button>
            ) : (
              <Button size="sm" onClick={handleAddNew}>
                <Plus className="mr-1.5 h-4 w-4" />
                Add Integration
              </Button>
            )}
          </div>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {filteredIntegrations.map((integration) => (
              <IntegrationCard
                key={integration.id}
                integration={integration}
                onViewDetails={handleViewDetails}
                onConfigure={handleConfigure}
                onSyncNow={triggerSync}
                onDisconnect={handleDisconnect}
                syncing={syncing}
              />
            ))}
          </div>
        )}

        {/* Filtered count */}
        {!isLoading && !error && filteredIntegrations.length > 0 && hasActiveFilters && (
          <p className="text-xs text-muted-foreground text-center">
            Showing {filteredIntegrations.length} of {integrations.length} integrations
          </p>
        )}

        {/* Detail Panel */}
        <IntegrationDetailPanel
          integration={selectedIntegration}
          open={detailOpen}
          onOpenChange={setDetailOpen}
          onSyncNow={triggerSync}
          onConfigure={handleConfigure}
          onDisconnect={handleDisconnect}
          syncing={syncing}
        />

        {/* Form Dialog */}
        <IntegrationFormDialog
          open={formOpen}
          onOpenChange={setFormOpen}
          integration={editIntegration}
        />

        {/* Disconnect Confirmation */}
        <ConfirmDialog
          open={disconnectOpen}
          onOpenChange={setDisconnectOpen}
          title="Disconnect Integration"
          description={`Are you sure you want to disconnect "${disconnectTarget?.name ?? ''}"? This will stop all data synchronization from this integration.`}
          confirmLabel="Disconnect"
          variant="destructive"
          onConfirm={confirmDisconnect}
        />
      </div>
    </PermissionRedirect>
  );
}
