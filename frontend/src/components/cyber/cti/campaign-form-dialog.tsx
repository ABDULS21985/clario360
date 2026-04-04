'use client';

import { useEffect, useMemo } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useForm } from 'react-hook-form';
import { z } from 'zod';
import { toast } from 'sonner';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { ScrollArea } from '@/components/ui/scroll-area';
import { FormField } from '@/components/shared/forms/form-field';
import { MultiSelect } from '@/components/shared/forms/multi-select';
import {
  createCampaign,
  fetchRegions,
  fetchSectors,
  fetchSeverityLevels,
  fetchThreatActors,
  updateCampaign,
} from '@/lib/cti-api';
import { buildTagInputValue, CTI_CAMPAIGN_STATUS_OPTIONS, parseTagInput } from '@/lib/cti-utils';
import type { CTICampaign, CTICampaignDetail, CTICampaignStatus, CTISeverityCode } from '@/types/cti';

const campaignSchema = z.object({
  campaign_code: z.string().max(50).optional(),
  name: z.string().min(1, 'Name is required').max(300),
  description: z.string().optional(),
  status: z.enum(['active', 'monitoring', 'dormant', 'resolved', 'archived']),
  severity_code: z.enum(['critical', 'high', 'medium', 'low', 'informational']),
  primary_actor_id: z.string().optional(),
  target_sectors: z.array(z.string()).default([]),
  target_regions: z.array(z.string()).default([]),
  target_description: z.string().optional(),
  mitre_technique_input: z.string().optional(),
  ttps_summary: z.string().optional(),
  first_seen_at: z.string().optional(),
});

type CampaignFormValues = z.infer<typeof campaignSchema>;

interface CampaignFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  campaign?: CTICampaign | CTICampaignDetail | null;
  onSuccess?: (campaign: CTICampaignDetail) => void;
}

function toDateTimeLocal(value?: string | null): string {
  if (!value) {
    return '';
  }

  const date = new Date(value);
  const offset = date.getTimezoneOffset() * 60_000;
  return new Date(date.getTime() - offset).toISOString().slice(0, 16);
}

function asCampaignStatus(value?: string | null): CTICampaignStatus {
  switch (value) {
    case 'active':
    case 'monitoring':
    case 'dormant':
    case 'resolved':
    case 'archived':
      return value;
    default:
      return 'monitoring';
  }
}

function asSeverityCode(value?: string | null): CTISeverityCode {
  switch (value) {
    case 'critical':
    case 'high':
    case 'medium':
    case 'low':
    case 'informational':
      return value;
    default:
      return 'medium';
  }
}

export function CampaignFormDialog({
  open,
  onOpenChange,
  campaign,
  onSuccess,
}: CampaignFormDialogProps) {
  const queryClient = useQueryClient();
  const isEditing = Boolean(campaign);
  const form = useForm<CampaignFormValues>({
    resolver: zodResolver(campaignSchema),
    defaultValues: {
      campaign_code: '',
      name: '',
      description: '',
      status: 'monitoring',
      severity_code: 'medium',
      primary_actor_id: '',
      target_sectors: [],
      target_regions: [],
      target_description: '',
      mitre_technique_input: '',
      ttps_summary: '',
      first_seen_at: '',
    },
  });

  const actorsQuery = useQuery({
    queryKey: ['cti-campaign-form-actors'],
    queryFn: () => fetchThreatActors({ page: 1, per_page: 100, sort: 'name', order: 'asc' }),
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

  useEffect(() => {
    if (!open) {
      return;
    }

    form.reset({
      campaign_code: campaign?.campaign_code ?? '',
      name: campaign?.name ?? '',
      description: campaign?.description ?? '',
      status: asCampaignStatus(campaign?.status),
      severity_code: asSeverityCode(campaign?.severity_code),
      primary_actor_id: campaign?.primary_actor_id ?? '',
      target_sectors: campaign?.target_sectors ?? [],
      target_regions: campaign?.target_regions ?? [],
      target_description: campaign?.target_description ?? '',
      mitre_technique_input: buildTagInputValue(campaign?.mitre_technique_ids),
      ttps_summary: campaign?.ttps_summary ?? '',
      first_seen_at: toDateTimeLocal(campaign?.first_seen_at),
    });
  }, [campaign, form, open]);

  const mutation = useMutation({
    mutationFn: async (values: CampaignFormValues) => {
      if (campaign) {
        return updateCampaign(campaign.id, {
          name: values.name.trim(),
          description: values.description?.trim() || undefined,
          severity_code: values.severity_code,
          primary_actor_id: values.primary_actor_id || undefined,
          target_description: values.target_description?.trim() || undefined,
          mitre_technique_ids: parseTagInput(values.mitre_technique_input ?? ''),
          ttps_summary: values.ttps_summary?.trim() || undefined,
        });
      }

      return createCampaign({
        campaign_code: values.campaign_code?.trim() || values.name.trim().toUpperCase().replace(/\s+/g, '-').slice(0, 50),
        name: values.name.trim(),
        description: values.description?.trim() || undefined,
        status: values.status,
        severity_code: values.severity_code,
        primary_actor_id: values.primary_actor_id || undefined,
        target_sectors: values.target_sectors,
        target_regions: values.target_regions,
        target_description: values.target_description?.trim() || undefined,
        mitre_technique_ids: parseTagInput(values.mitre_technique_input ?? ''),
        ttps_summary: values.ttps_summary?.trim() || undefined,
        first_seen_at: values.first_seen_at ? new Date(values.first_seen_at).toISOString() : new Date().toISOString(),
      });
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

  const onSubmit = form.handleSubmit((values) => mutation.mutate(values));

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>{campaign ? 'Edit Campaign' : 'Create Campaign'}</DialogTitle>
          <DialogDescription>
            Manage campaign metadata, targeting, and primary actor assignment against the live CTI API.
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <form onSubmit={onSubmit} className="space-y-5">
            <ScrollArea className="max-h-[70vh] pr-4">
              <div className="space-y-5">
                <div className="grid gap-4 md:grid-cols-2">
                  {!isEditing && (
                    <FormField name="campaign_code" label="Campaign Code" required>
                      <Input id="campaign_code" placeholder="WINTER-OAUTH-SPRAY" {...form.register('campaign_code')} />
                    </FormField>
                  )}
                  <FormField name="name" label="Campaign Name" required className={!isEditing ? '' : 'md:col-span-2'}>
                    <Input id="name" placeholder="Winter OAuth Spray" {...form.register('name')} />
                  </FormField>
                  {!isEditing && (
                    <FormField name="status" label="Status" required>
                      <Select
                        value={form.watch('status')}
                        onValueChange={(value) => form.setValue('status', value as CampaignFormValues['status'])}
                      >
                        <SelectTrigger id="status">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {CTI_CAMPAIGN_STATUS_OPTIONS.map((option) => (
                            <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </FormField>
                  )}
                  <FormField name="severity_code" label="Severity" required>
                    <Select
                      value={form.watch('severity_code')}
                      onValueChange={(value) => form.setValue('severity_code', value as CampaignFormValues['severity_code'])}
                    >
                      <SelectTrigger id="severity_code">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {severityOptions.map((option) => (
                          <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </FormField>
                  <FormField name="primary_actor_id" label="Primary Actor">
                    <Select
                      value={form.watch('primary_actor_id') || 'none'}
                      onValueChange={(value) => form.setValue('primary_actor_id', value === 'none' ? '' : value)}
                    >
                      <SelectTrigger id="primary_actor_id">
                        <SelectValue placeholder="No primary actor" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="none">No primary actor</SelectItem>
                        {actorOptions.map((option) => (
                          <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </FormField>
                  {!isEditing && (
                    <FormField name="first_seen_at" label="First Seen" required className="md:col-span-2">
                      <Input id="first_seen_at" type="datetime-local" {...form.register('first_seen_at')} />
                    </FormField>
                  )}
                </div>
                <FormField name="description" label="Description">
                  <Textarea id="description" rows={4} {...form.register('description')} />
                </FormField>
                {!isEditing && (
                  <div className="grid gap-4 md:grid-cols-2">
                    <FormField name="target_sectors" label="Target Sectors">
                      <MultiSelect
                        options={sectorOptions}
                        selected={form.watch('target_sectors')}
                        onChange={(values) => form.setValue('target_sectors', values, { shouldValidate: true })}
                        placeholder="Select sectors"
                      />
                    </FormField>
                    <FormField name="target_regions" label="Target Regions">
                      <MultiSelect
                        options={regionOptions}
                        selected={form.watch('target_regions')}
                        onChange={(values) => form.setValue('target_regions', values, { shouldValidate: true })}
                        placeholder="Select regions"
                      />
                    </FormField>
                  </div>
                )}
                <FormField name="target_description" label="Targeting Notes">
                  <Textarea id="target_description" rows={3} {...form.register('target_description')} />
                </FormField>
                <FormField name="mitre_technique_input" label="MITRE Techniques">
                  <Input
                    id="mitre_technique_input"
                    placeholder="T1078, T1566.002"
                    {...form.register('mitre_technique_input')}
                  />
                </FormField>
                <FormField name="ttps_summary" label="TTP Summary">
                  <Textarea id="ttps_summary" rows={4} {...form.register('ttps_summary')} />
                </FormField>
              </div>
            </ScrollArea>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={mutation.isPending}>
                {mutation.isPending ? 'Saving...' : campaign ? 'Save Changes' : 'Create Campaign'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
