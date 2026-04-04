'use client';

import { useMemo } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { CampaignForm, type CampaignFormPayload } from '@/components/cyber/cti/campaign-form';
import {
  createCampaign,
  fetchRegions,
  fetchSectors,
  fetchSeverityLevels,
  fetchThreatActors,
  updateCampaign,
  updateCampaignStatus,
} from '@/lib/cti-api';
import type { CTICampaign, CTICampaignDetail, CreateCampaignRequest } from '@/types/cti';

interface CampaignFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  campaign?: CTICampaign | CTICampaignDetail | null;
  onSuccess?: (campaign: CTICampaignDetail) => void;
}

export function CampaignFormDialog({
  open,
  onOpenChange,
  campaign,
  onSuccess,
}: CampaignFormDialogProps) {
  const queryClient = useQueryClient();

  const actorsQuery = useQuery({
    queryKey: ['cti-campaign-form-actors'],
    queryFn: () => fetchThreatActors({ page: 1, per_page: 200, sort: 'name', order: 'asc' }),
    enabled: open,
  });
  const sectorsQuery = useQuery({
    queryKey: ['cti-campaign-form-sectors'],
    queryFn: fetchSectors,
    enabled: open,
  });
  const regionsQuery = useQuery({
    queryKey: ['cti-campaign-form-regions'],
    queryFn: () => fetchRegions(),
    enabled: open,
  });
  const severityQuery = useQuery({
    queryKey: ['cti-campaign-form-severities'],
    queryFn: fetchSeverityLevels,
    enabled: open,
  });

  const actorOptions = useMemo(
    () => (actorsQuery.data?.data ?? []).map((actor) => ({ label: actor.name, value: actor.id })),
    [actorsQuery.data],
  );
  const sectorOptions = useMemo(
    () => (sectorsQuery.data ?? []).map((sector) => ({ label: sector.label, value: sector.id })),
    [sectorsQuery.data],
  );
  const regionOptions = useMemo(
    () => (regionsQuery.data ?? []).map((region) => ({ label: region.label, value: region.id })),
    [regionsQuery.data],
  );
  const severityOptions = useMemo(
    () => (severityQuery.data ?? []).map((severity) => ({ label: severity.label, value: severity.code })),
    [severityQuery.data],
  );

  const mutation = useMutation({
    mutationFn: async (values: CreateCampaignRequest | CampaignFormPayload) => {
      if (campaign) {
        const nextStatus = 'status' in values && values.status ? values.status : campaign.status;
        const updated = await updateCampaign(campaign.id, values);
        if (nextStatus !== campaign.status) {
          await updateCampaignStatus(campaign.id, nextStatus);
        }
        return 'status' in values && nextStatus !== campaign.status
          ? { ...updated, status: nextStatus }
          : updated;
      }

      return createCampaign(values as CreateCampaignRequest);
    },
    onSuccess: async (savedCampaign) => {
      await queryClient.invalidateQueries({ queryKey: ['cti-campaigns'] });
      if (campaign) {
        await queryClient.invalidateQueries({ queryKey: ['cti-campaign', campaign.id] });
      }
      toast.success(campaign ? 'Campaign updated' : 'Campaign created');
      onOpenChange(false);
      onSuccess?.(savedCampaign);
    },
    onError: () => {
      toast.error(campaign ? 'Failed to update campaign' : 'Failed to create campaign');
    },
  });

  const handleSubmit = async (values: CreateCampaignRequest | CampaignFormPayload) => {
    await mutation.mutateAsync(values);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>{campaign ? 'Edit Campaign' : 'Create Campaign'}</DialogTitle>
          <DialogDescription>
            Manage campaign metadata, targeting, and primary actor assignment against the live CTI API.
          </DialogDescription>
        </DialogHeader>
        <CampaignForm
          campaign={campaign}
          actors={actorOptions}
          sectors={sectorOptions}
          regions={regionOptions}
          severities={severityOptions}
          onSubmit={handleSubmit}
          onCancel={() => onOpenChange(false)}
          isLoading={mutation.isPending}
        />
      </DialogContent>
    </Dialog>
  );
}
