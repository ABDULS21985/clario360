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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { FormField } from '@/components/shared/forms/form-field';
import {
  createBrandAbuseIncident,
  fetchMonitoredBrands,
  updateBrandAbuseIncident,
} from '@/lib/cti-api';
import { CTI_BRAND_ABUSE_TYPE_OPTIONS, CTI_RISK_LEVEL_OPTIONS } from '@/lib/cti-utils';
import type { CTIBrandAbuseIncident } from '@/types/cti';

const incidentSchema = z.object({
  brand_id: z.string().min(1, 'A monitored brand is required'),
  malicious_domain: z.string().min(1, 'Malicious domain is required').max(500),
  abuse_type: z.string().min(1, 'Abuse type is required'),
  risk_level: z.enum(['critical', 'high', 'medium', 'low']),
  source_name: z.string().optional(),
  whois_registrant: z.string().optional(),
  whois_created_date: z.string().optional(),
  ssl_issuer: z.string().optional(),
  hosting_ip: z.string().optional(),
  hosting_asn: z.string().optional(),
  detection_count: z.coerce.number().min(1).optional(),
});

type IncidentFormValues = z.infer<typeof incidentSchema>;

interface BrandAbuseFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  incident?: CTIBrandAbuseIncident | null;
  onSuccess?: (incident?: CTIBrandAbuseIncident) => void;
}

export function BrandAbuseFormDialog({
  open,
  onOpenChange,
  incident,
  onSuccess,
}: BrandAbuseFormDialogProps) {
  const queryClient = useQueryClient();
  const form = useForm<IncidentFormValues>({
    resolver: zodResolver(incidentSchema),
    defaultValues: {
      brand_id: '',
      malicious_domain: '',
      abuse_type: 'typosquatting',
      risk_level: 'medium',
      source_name: '',
      whois_registrant: '',
      whois_created_date: '',
      ssl_issuer: '',
      hosting_ip: '',
      hosting_asn: '',
      detection_count: 1,
    },
  });

  const brandsQuery = useQuery({
    queryKey: ['cti-brand-abuse-form-brands'],
    queryFn: fetchMonitoredBrands,
    enabled: open,
  });
  const brandOptions = useMemo(
    () => (brandsQuery.data ?? []).map((brand) => ({ label: brand.brand_name, value: brand.id })),
    [brandsQuery.data],
  );

  useEffect(() => {
    if (!open) {
      return;
    }

    form.reset({
      brand_id: incident?.brand_id ?? '',
      malicious_domain: incident?.malicious_domain ?? '',
      abuse_type: incident?.abuse_type ?? 'typosquatting',
      risk_level: incident?.risk_level ?? 'medium',
      source_name: incident?.source_id ?? '',
      whois_registrant: incident?.whois_registrant ?? '',
      whois_created_date: incident?.whois_created_date ?? '',
      ssl_issuer: incident?.ssl_issuer ?? '',
      hosting_ip: incident?.hosting_ip ?? '',
      hosting_asn: incident?.hosting_asn ?? '',
      detection_count: incident?.detection_count ?? 1,
    });
  }, [form, incident, open]);

  const mutation = useMutation({
    mutationFn: async (values: IncidentFormValues) => {
      if (incident) {
        await updateBrandAbuseIncident(incident.id, {
          brand_id: values.brand_id,
          malicious_domain: values.malicious_domain.trim(),
          abuse_type: values.abuse_type,
          risk_level: values.risk_level,
          source_name: values.source_name?.trim() || undefined,
          whois_registrant: values.whois_registrant?.trim() || undefined,
          whois_created_date: values.whois_created_date?.trim() || undefined,
          ssl_issuer: values.ssl_issuer?.trim() || undefined,
          hosting_ip: values.hosting_ip?.trim() || undefined,
          hosting_asn: values.hosting_asn?.trim() || undefined,
          detection_count: values.detection_count,
        });
        return incident;
      }

      return createBrandAbuseIncident({
        brand_id: values.brand_id,
        malicious_domain: values.malicious_domain.trim(),
        abuse_type: values.abuse_type,
        risk_level: values.risk_level,
        source_name: values.source_name?.trim() || undefined,
        whois_registrant: values.whois_registrant?.trim() || undefined,
        whois_created_date: values.whois_created_date?.trim() || undefined,
        ssl_issuer: values.ssl_issuer?.trim() || undefined,
        hosting_ip: values.hosting_ip?.trim() || undefined,
        hosting_asn: values.hosting_asn?.trim() || undefined,
      });
    },
    onSuccess: async (savedIncident) => {
      await queryClient.invalidateQueries({ queryKey: ['cti-brand-abuse'] });
      await queryClient.invalidateQueries({ queryKey: ['cti-brands'] });
      if (incident) {
        await queryClient.invalidateQueries({ queryKey: ['cti-brand-abuse-incident', incident.id] });
      }
      toast.success(incident ? 'Brand abuse incident updated' : 'Brand abuse incident created');
      onOpenChange(false);
      onSuccess?.(savedIncident);
    },
    onError: () => {
      toast.error(incident ? 'Failed to update brand abuse incident' : 'Failed to create brand abuse incident');
    },
  });

  const onSubmit = form.handleSubmit((values) => mutation.mutate(values));

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{incident ? 'Edit Brand Abuse Incident' : 'Create Brand Abuse Incident'}</DialogTitle>
          <DialogDescription>
            Track malicious domains, takedown posture, and registrar or hosting clues for brand monitoring.
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <form onSubmit={onSubmit} className="space-y-4">
            <div className="grid gap-4 md:grid-cols-2">
              <FormField name="brand_id" label="Monitored Brand" required className="md:col-span-2">
                <Select
                  value={form.watch('brand_id')}
                  onValueChange={(value) => form.setValue('brand_id', value, { shouldValidate: true })}
                >
                  <SelectTrigger id="brand_id">
                    <SelectValue placeholder="Select a monitored brand" />
                  </SelectTrigger>
                  <SelectContent>
                    {brandOptions.map((option) => (
                      <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
              <FormField name="malicious_domain" label="Malicious Domain" required className="md:col-span-2">
                <Input id="malicious_domain" placeholder="secure-clario-support[.]com" {...form.register('malicious_domain')} />
              </FormField>
              <FormField name="abuse_type" label="Abuse Type" required>
                <Select
                  value={form.watch('abuse_type')}
                  onValueChange={(value) => form.setValue('abuse_type', value)}
                >
                  <SelectTrigger id="abuse_type">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {CTI_BRAND_ABUSE_TYPE_OPTIONS.map((option) => (
                      <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
              <FormField name="risk_level" label="Risk Level" required>
                <Select
                  value={form.watch('risk_level')}
                  onValueChange={(value) => form.setValue('risk_level', value as IncidentFormValues['risk_level'])}
                >
                  <SelectTrigger id="risk_level">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {CTI_RISK_LEVEL_OPTIONS.map((option) => (
                      <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
              <FormField name="source_name" label="Source Name">
                <Input id="source_name" placeholder="Abuse mailbox" {...form.register('source_name')} />
              </FormField>
              <FormField name="whois_registrant" label="WHOIS Registrant">
                <Input id="whois_registrant" {...form.register('whois_registrant')} />
              </FormField>
              <FormField name="whois_created_date" label="WHOIS Created Date">
                <Input id="whois_created_date" placeholder="2024-11-28" {...form.register('whois_created_date')} />
              </FormField>
              <FormField name="ssl_issuer" label="SSL Issuer">
                <Input id="ssl_issuer" {...form.register('ssl_issuer')} />
              </FormField>
              <FormField name="hosting_ip" label="Hosting IP">
                <Input id="hosting_ip" {...form.register('hosting_ip')} />
              </FormField>
              <FormField name="hosting_asn" label="Hosting ASN">
                <Input id="hosting_asn" {...form.register('hosting_asn')} />
              </FormField>
              {incident && (
                <FormField name="detection_count" label="Detection Count">
                  <Input id="detection_count" type="number" min={1} {...form.register('detection_count')} />
                </FormField>
              )}
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={mutation.isPending}>
                {mutation.isPending ? 'Saving...' : incident ? 'Save Changes' : 'Create Incident'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}