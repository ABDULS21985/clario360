'use client';

import { useMemo, useState } from 'react';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, ChevronDown, Pencil } from 'lucide-react';
import { toast } from 'sonner';
import { PageHeader } from '@/components/common/page-header';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { PermissionGate } from '@/components/auth/permission-gate';
import { BrandAbuseFormDialog } from '@/components/cyber/cti/brand-abuse-form-dialog';
import { IOCValueDisplay } from '@/components/cyber/cti/ioc-value-display';
import { MonitoredBrandsManager } from '@/components/cyber/cti/monitored-brands-manager';
import { CTIKPIStatCard } from '@/components/cyber/cti/kpi-stat-card';
import { CTISeverityBadge } from '@/components/cyber/cti/severity-badge';
import { CTIStatusBadge } from '@/components/cyber/cti/status-badge';
import { TakedownPipeline } from '@/components/cyber/cti/takedown-pipeline';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { fetchBrandAbuseIncident, fetchMonitoredBrands, updateTakedownStatus } from '@/lib/cti-api';
import { CTI_TAKEDOWN_STATUS_OPTIONS } from '@/lib/cti-utils';
import { formatDateTime } from '@/lib/utils';
import { ROUTES } from '@/lib/constants';

export default function CTIBrandAbuseDetailPage() {
  const params = useParams<{ id: string }>();
  const incidentId = params?.id ?? '';
  const router = useRouter();
  const queryClient = useQueryClient();
  const [editOpen, setEditOpen] = useState(false);
  const [brandsManagerOpen, setBrandsManagerOpen] = useState(false);

  const incidentQuery = useQuery({
    queryKey: ['cti-brand-abuse-incident', incidentId],
    queryFn: () => fetchBrandAbuseIncident(incidentId),
    enabled: Boolean(incidentId),
  });
  const brandsQuery = useQuery({
    queryKey: ['cti-brand-abuse-detail-brands'],
    queryFn: fetchMonitoredBrands,
  });

  const incident = incidentQuery.data;
  const monitoredBrand = useMemo(
    () => brandsQuery.data?.find((brand) => brand.id === incident?.brand_id) ?? null,
    [brandsQuery.data, incident?.brand_id],
  );

  const statusMutation = useMutation({
    mutationFn: async (status: string) => updateTakedownStatus(incidentId, status),
    onMutate: async (status) => {
      const previousIncident = queryClient.getQueryData(['cti-brand-abuse-incident', incidentId]);
      queryClient.setQueryData(['cti-brand-abuse-incident', incidentId], (current: typeof incident) => (
        current ? { ...current, takedown_status: status } : current
      ));
      return { previousIncident };
    },
    onError: (_error, _status, context) => {
      queryClient.setQueryData(['cti-brand-abuse-incident', incidentId], context?.previousIncident);
      toast.error('Failed to update takedown status');
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['cti-brand-abuse-incident', incidentId] }),
        queryClient.invalidateQueries({ queryKey: ['cti-brand-abuse'] }),
      ]);
      toast.success('Takedown status updated');
    },
  });

  if (incidentQuery.isLoading) {
    return (
      <PermissionRedirect permission="cyber:read">
        <LoadingSkeleton variant="card" count={2} />
      </PermissionRedirect>
    );
  }

  if (!incident || incidentQuery.error) {
    return (
      <PermissionRedirect permission="cyber:read">
        <ErrorState message="Failed to load brand abuse incident" onRetry={() => void incidentQuery.refetch()} />
      </PermissionRedirect>
    );
  }

  const daysActive = Math.max(
    Math.ceil((Date.parse(incident.last_detected_at) - Date.parse(incident.first_detected_at)) / 86_400_000),
    1,
  );

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title={(
            <div className="flex items-center gap-3">
              <button
                onClick={() => router.push(ROUTES.CYBER_CTI_BRAND_ABUSE)}
                className="flex h-8 w-8 items-center justify-center rounded-full border bg-background text-muted-foreground shadow-sm transition-colors hover:bg-accent"
              >
                <ArrowLeft className="h-4 w-4" />
              </button>
              <span className="truncate">{incident.malicious_domain}</span>
            </div>
          )}
          description={(
            <div className="flex flex-wrap items-center gap-3 pl-11">
              <span className="text-sm text-muted-foreground">{incident.brand_name}</span>
              <CTISeverityBadge severity={incident.risk_level} />
              <CTIStatusBadge status={incident.takedown_status} type="takedown" />
              <span className="rounded-full border px-3 py-1 text-xs font-medium text-muted-foreground">
                {incident.abuse_type.replaceAll('_', ' ')}
              </span>
            </div>
          )}
          actions={(
            <div className="flex flex-wrap items-center gap-2">
              <PermissionGate permission="cyber:write">
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="outline" size="sm">
                      Update Status
                      <ChevronDown className="ml-1.5 h-3.5 w-3.5" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    {CTI_TAKEDOWN_STATUS_OPTIONS.filter((option) => option.value !== incident.takedown_status).map((option) => (
                      <DropdownMenuItem key={option.value} onClick={() => statusMutation.mutate(option.value)}>
                        Move to {option.label}
                      </DropdownMenuItem>
                    ))}
                  </DropdownMenuContent>
                </DropdownMenu>
                <Button variant="outline" size="sm" onClick={() => setBrandsManagerOpen(true)}>
                  Manage Brands
                </Button>
                <Button variant="outline" size="sm" onClick={() => setEditOpen(true)}>
                  <Pencil className="mr-1.5 h-3.5 w-3.5" />
                  Edit
                </Button>
              </PermissionGate>
            </div>
          )}
        />

        <div className="grid gap-4 md:grid-cols-3">
          <CTIKPIStatCard label="Detection Count" value={incident.detection_count} subtitle="Observed detections" />
          <CTIKPIStatCard label="Takedown Status" value={incident.takedown_status.replaceAll('_', ' ')} subtitle="Current lifecycle stage" />
          <CTIKPIStatCard label="Days Active" value={daysActive} subtitle="Time since first sighting" />
        </div>

        <TakedownPipeline
          status={incident.takedown_status}
          requestedAt={incident.takedown_requested_at}
          takenDownAt={incident.taken_down_at}
          onAdvanceStatus={(nextStatus) => statusMutation.mutate(nextStatus)}
          isLoading={statusMutation.isPending}
        />

        <div className="flex flex-wrap gap-2">
          <PermissionGate permission="cyber:write">
            {incident.takedown_status !== 'takedown_requested' && incident.takedown_status !== 'taken_down' && (
              <Button variant="outline" onClick={() => statusMutation.mutate('takedown_requested')}>
                Request Takedown
              </Button>
            )}
            {incident.takedown_status !== 'taken_down' && (
              <Button variant="outline" onClick={() => statusMutation.mutate('taken_down')}>
                Mark Taken Down
              </Button>
            )}
            {incident.takedown_status !== 'false_positive' && (
              <Button variant="outline" onClick={() => statusMutation.mutate('false_positive')}>
                Mark False Positive
              </Button>
            )}
          </PermissionGate>
          <Button variant="ghost" asChild>
            <Link href={`${ROUTES.CYBER_CTI_EVENTS}?search=${encodeURIComponent(incident.malicious_domain)}`}>
              Related Events →
            </Link>
          </Button>
        </div>

        <div className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Incident Details</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              <IOCValueDisplay type="domain" value={incident.malicious_domain} />
              <DetailRow label="Abuse Type" value={incident.abuse_type.replaceAll('_', ' ')} />
              <DetailRow label="Region" value={incident.region_label || '—'} />
              <DetailRow label="Hosting IP" value={incident.hosting_ip || '—'} />
              <DetailRow label="Hosting ASN" value={incident.hosting_asn || '—'} />
              <DetailRow label="WHOIS Registrant" value={incident.whois_registrant || '—'} />
              <DetailRow label="WHOIS Created" value={incident.whois_created_date || '—'} />
              <DetailRow label="SSL Issuer" value={incident.ssl_issuer || '—'} />
              <DetailRow label="First Detected" value={formatDateTime(incident.first_detected_at)} />
              <DetailRow label="Last Detected" value={formatDateTime(incident.last_detected_at)} />
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>Monitored Brand Context</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              <DetailRow label="Brand" value={incident.brand_name} />
              <DetailRow label="Domain Pattern" value={monitoredBrand?.domain_pattern || '—'} />
              <div>
                <p className="mb-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                  Keywords
                </p>
                <div className="flex flex-wrap gap-2">
                  {(monitoredBrand?.keywords ?? []).length > 0 ? monitoredBrand!.keywords.map((keyword) => (
                    <span key={keyword} className="rounded-full border px-3 py-1 text-sm text-muted-foreground">{keyword}</span>
                  )) : (
                    <span className="text-muted-foreground">No keywords configured</span>
                  )}
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      <BrandAbuseFormDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        incident={incident}
        onSuccess={() => void incidentQuery.refetch()}
      />

      <MonitoredBrandsManager open={brandsManagerOpen} onOpenChange={setBrandsManagerOpen} />
    </PermissionRedirect>
  );
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-4">
      <span className="text-muted-foreground">{label}</span>
      <span className="text-right font-medium text-foreground">{value || '—'}</span>
    </div>
  );
}
