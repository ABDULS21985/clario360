'use client';

import { useState } from 'react';
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
import { fetchBrandAbuseIncident, updateTakedownStatus } from '@/lib/cti-api';
import { CTI_TAKEDOWN_STATUS_OPTIONS, formatCountryCode } from '@/lib/cti-utils';
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

  const incident = incidentQuery.data;

  const statusMutation = useMutation({
    mutationFn: async (status: string) => updateTakedownStatus(incidentId, status),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['cti-brand-abuse-incident', incidentId] }),
        queryClient.invalidateQueries({ queryKey: ['cti-brand-abuse'] }),
      ]);
      toast.success('Takedown status updated');
    },
    onError: () => toast.error('Failed to update takedown status'),
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

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title={
            <div className="flex items-center gap-3">
              <button
                onClick={() => router.push(ROUTES.CYBER_CTI_BRAND_ABUSE)}
                className="flex h-8 w-8 items-center justify-center rounded-full border bg-background text-muted-foreground shadow-sm transition-colors hover:bg-accent"
              >
                <ArrowLeft className="h-4 w-4" />
              </button>
              <span className="truncate">{incident.brand_name}</span>
            </div>
          }
          description={
            <div className="flex flex-wrap items-center gap-3 pl-11">
              <CTISeverityBadge severity={incident.risk_level} />
              <CTIStatusBadge status={incident.takedown_status} type="takedown" />
              <span className="rounded-full border px-3 py-1 text-xs font-medium text-muted-foreground">
                {incident.abuse_type.replaceAll('_', ' ')}
              </span>
            </div>
          }
          actions={
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
          }
        />

        <TakedownPipeline
          status={incident.takedown_status}
          requestedAt={incident.takedown_requested_at}
          takenDownAt={incident.taken_down_at}
        />

        <div className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Abuse Details</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              <IOCValueDisplay type="domain" value={incident.malicious_domain} />
              <DetailRow label="Detection Count" value={String(incident.detection_count)} />
              <DetailRow label="Region" value={incident.region_label || '—'} />
              <DetailRow label="Hosting ASN" value={incident.hosting_asn || '—'} />
              <DetailRow label="Hosting IP" value={incident.hosting_ip || '—'} />
              <DetailRow label="First Detected" value={formatDateTime(incident.first_detected_at)} />
              <DetailRow label="Last Detected" value={formatDateTime(incident.last_detected_at)} />
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>Infrastructure Clues</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              <DetailRow label="WHOIS Registrant" value={incident.whois_registrant || '—'} />
              <DetailRow label="WHOIS Created" value={incident.whois_created_date || '—'} />
              <DetailRow label="SSL Issuer" value={incident.ssl_issuer || '—'} />
              <DetailRow label="Source Ref" value={incident.source_id || '—'} />
              <DetailRow label="Region Code" value={formatCountryCode(incident.region_id)} />
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