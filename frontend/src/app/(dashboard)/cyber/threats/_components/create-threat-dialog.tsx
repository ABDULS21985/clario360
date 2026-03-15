'use client';

import { useEffect, useMemo } from 'react';
import { useFieldArray, useForm, FormProvider } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';
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
import { Slider } from '@/components/ui/slider';
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
import { useApiMutation } from '@/hooks/use-api-mutation';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import {
  emptyIndicator,
  INDICATOR_TYPE_OPTIONS,
  THREAT_TYPE_OPTIONS,
} from '@/lib/cyber-threats';
import type {
  CreateThreatInput,
  MITRETacticItem,
  MITRETechniqueItem,
  Threat,
} from '@/types/cyber';

const threatSchema = z.object({
  name: z.string().min(1, 'Name is required').max(255),
  type: z.enum([
    'malware',
    'phishing',
    'apt',
    'ransomware',
    'ddos',
    'insider_threat',
    'supply_chain',
    'zero_day',
    'brute_force',
    'other',
  ]),
  severity: z.enum(['critical', 'high', 'medium', 'low']),
  description: z.string().optional(),
  threat_actor: z.string().optional(),
  campaign: z.string().optional(),
  mitre_tactic_ids: z.array(z.string()).default([]),
  mitre_technique_ids: z.array(z.string()).default([]),
  tags_input: z.string().optional(),
  indicators: z.array(z.object({
    type: z.enum([
      'ip',
      'domain',
      'url',
      'email',
      'file_hash_md5',
      'file_hash_sha1',
      'file_hash_sha256',
      'certificate',
      'registry_key',
      'user_agent',
      'cidr',
    ]),
    value: z.string().min(1, 'Indicator value is required'),
    severity: z.enum(['critical', 'high', 'medium', 'low']),
    confidence: z.number().min(0).max(100),
    description: z.string().optional(),
  })).default([]),
});

type ThreatFormValues = z.infer<typeof threatSchema>;

interface CreateThreatDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  threat?: Threat | null;
  onSuccess?: (threat: Threat) => void;
}

export function CreateThreatDialog({
  open,
  onOpenChange,
  threat,
  onSuccess,
}: CreateThreatDialogProps) {
  const isEditing = Boolean(threat);
  const methods = useForm<ThreatFormValues>({
    resolver: zodResolver(threatSchema),
    defaultValues: {
      name: '',
      type: 'malware',
      severity: 'medium',
      description: '',
      threat_actor: '',
      campaign: '',
      mitre_tactic_ids: [],
      mitre_technique_ids: [],
      tags_input: '',
      indicators: [],
    },
  });

  const { fields, append, remove } = useFieldArray({
    control: methods.control,
    name: 'indicators',
  });

  const { data: tacticsEnvelope } = useQuery({
    queryKey: ['mitre-tactics'],
    queryFn: () => apiGet<{ data: MITRETacticItem[] }>(API_ENDPOINTS.CYBER_MITRE_TACTICS),
    staleTime: 300000,
  });

  const selectedTactics = methods.watch('mitre_tactic_ids');
  const techniquesQuery = useQuery({
    queryKey: ['mitre-techniques', selectedTactics],
    queryFn: async () => {
      if (selectedTactics.length === 0) {
        return apiGet<{ data: MITRETechniqueItem[] }>(API_ENDPOINTS.CYBER_MITRE_TECHNIQUES);
      }
      const all = await Promise.all(
        selectedTactics.map((id) =>
          apiGet<{ data: MITRETechniqueItem[] }>(API_ENDPOINTS.CYBER_MITRE_TECHNIQUES, { tactic_id: id }),
        ),
      );
      return {
        data: all.flatMap((entry) => entry.data).filter((item, index, arr) => (
          arr.findIndex((candidate) => candidate.id === item.id) === index
        )),
      };
    },
    enabled: open,
    staleTime: 300000,
  });

  const tacticOptions = useMemo(
    () => (tacticsEnvelope?.data ?? []).map((item) => ({ label: `${item.id} · ${item.name}`, value: item.id })),
    [tacticsEnvelope],
  );
  const techniqueOptions = useMemo(
    () => (techniquesQuery.data?.data ?? []).map((item) => ({ label: `${item.id} · ${item.name}`, value: item.id })),
    [techniquesQuery.data],
  );

  useEffect(() => {
    if (!open) {
      return;
    }
    methods.reset({
      name: threat?.name ?? '',
      type: threat?.type ?? 'malware',
      severity: threat?.severity ?? 'medium',
      description: threat?.description ?? '',
      threat_actor: threat?.threat_actor ?? '',
      campaign: threat?.campaign ?? '',
      mitre_tactic_ids: threat?.mitre_tactic_ids ?? [],
      mitre_technique_ids: threat?.mitre_technique_ids ?? [],
      tags_input: (threat?.tags ?? []).join(', '),
      indicators: [],
    });
  }, [methods, open, threat]);

  useEffect(() => {
    const allowed = new Set(techniqueOptions.map((option) => option.value));
    const next = methods.getValues('mitre_technique_ids').filter((id) => allowed.has(id));
    if (next.length !== methods.getValues('mitre_technique_ids').length) {
      methods.setValue('mitre_technique_ids', next);
    }
  }, [methods, techniqueOptions]);

  const createMutation = useApiMutation<{ data: Threat }, CreateThreatInput>(
    'post',
    API_ENDPOINTS.CYBER_THREATS,
    {
      invalidateKeys: ['cyber-threats'],
      successMessage: 'Threat created',
      onSuccess: (response) => {
        methods.reset();
        onOpenChange(false);
        onSuccess?.(response.data);
      },
    },
  );

  const updateMutation = useApiMutation<{ data: Threat }, CreateThreatInput>(
    'put',
    () => API_ENDPOINTS.CYBER_THREAT_DETAIL(threat!.id),
    {
      invalidateKeys: ['cyber-threats', threat ? `cyber-threat-${threat.id}` : 'cyber-threats'],
      successMessage: 'Threat updated',
      onSuccess: (response) => {
        onOpenChange(false);
        onSuccess?.(response.data);
      },
    },
  );

  const isSubmitting = createMutation.isPending || updateMutation.isPending;

  const onSubmit = methods.handleSubmit((values) => {
    const payload: CreateThreatInput = {
      name: values.name.trim(),
      type: values.type,
      severity: values.severity,
      description: values.description?.trim() || undefined,
      threat_actor: values.threat_actor?.trim() || undefined,
      campaign: values.campaign?.trim() || undefined,
      mitre_tactic_ids: values.mitre_tactic_ids,
      mitre_technique_ids: values.mitre_technique_ids,
      tags: parseTags(values.tags_input),
      indicators: isEditing ? undefined : values.indicators.map((indicator) => ({
        ...indicator,
        value: indicator.value.trim(),
        description: indicator.description?.trim() || undefined,
        confidence: indicator.confidence / 100,
        source: 'manual',
      })),
    };

    if (isEditing) {
      updateMutation.mutate(payload);
      return;
    }
    createMutation.mutate(payload);
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>{isEditing ? 'Edit Threat' : 'Create Threat'}</DialogTitle>
          <DialogDescription>
            {isEditing
              ? 'Update the lifecycle, MITRE mapping, and analyst context for this threat.'
              : 'Capture a new threat, classify it, and attach the first indicators of compromise.'}
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...methods}>
          <form onSubmit={onSubmit} className="space-y-5">
            <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
              <FormField name="name" label="Name" required className="md:col-span-3">
                <Input id="name" placeholder="APT29 credential harvesting cluster" {...methods.register('name')} />
              </FormField>
              <FormField name="type" label="Type" required>
                <Select
                  value={methods.watch('type')}
                  onValueChange={(value) => methods.setValue('type', value as ThreatFormValues['type'])}
                >
                  <SelectTrigger id="type">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {THREAT_TYPE_OPTIONS.map((option) => (
                      <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
              <FormField name="severity" label="Severity" required>
                <Select
                  value={methods.watch('severity')}
                  onValueChange={(value) => methods.setValue('severity', value as ThreatFormValues['severity'])}
                >
                  <SelectTrigger id="severity">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {['critical', 'high', 'medium', 'low'].map((option) => (
                      <SelectItem key={option} value={option} className="capitalize">{option}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
              <FormField name="tags_input" label="Tags">
                <Input id="tags_input" placeholder="apt29, oauth, credential-access" {...methods.register('tags_input')} />
              </FormField>
            </div>

            <FormField name="description" label="Description">
              <Textarea
                id="description"
                rows={4}
                placeholder="Summarize the campaign, suspected intent, and observed behavior."
                {...methods.register('description')}
              />
            </FormField>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <FormField name="threat_actor" label="Threat Actor">
                <Input id="threat_actor" placeholder="APT29" {...methods.register('threat_actor')} />
              </FormField>
              <FormField name="campaign" label="Campaign">
                <Input id="campaign" placeholder="winter-oauth-spray" {...methods.register('campaign')} />
              </FormField>
            </div>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <FormField name="mitre_tactic_ids" label="MITRE Tactics">
                <MultiSelect
                  options={tacticOptions}
                  selected={methods.watch('mitre_tactic_ids')}
                  onChange={(values) => methods.setValue('mitre_tactic_ids', values, { shouldValidate: true })}
                  placeholder="Select tactics"
                />
              </FormField>
              <FormField name="mitre_technique_ids" label="MITRE Techniques">
                <MultiSelect
                  options={techniqueOptions}
                  selected={methods.watch('mitre_technique_ids')}
                  onChange={(values) => methods.setValue('mitre_technique_ids', values, { shouldValidate: true })}
                  placeholder={selectedTactics.length > 0 ? 'Select techniques' : 'Select tactics first'}
                  disabled={selectedTactics.length === 0}
                />
              </FormField>
            </div>

            {!isEditing && (
              <div className="rounded-2xl border border-border/70 bg-muted/20 p-4">
                <div className="mb-4 flex items-center justify-between gap-3">
                  <div>
                    <h3 className="text-sm font-semibold">Initial Indicators</h3>
                    <p className="text-xs text-muted-foreground">
                      Seed the threat with the IOCs analysts already have. You can add more later.
                    </p>
                  </div>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => append(emptyIndicator())}
                  >
                    Add Indicator
                  </Button>
                </div>

                {fields.length === 0 ? (
                  <div className="rounded-xl border border-dashed px-4 py-6 text-sm text-muted-foreground">
                    No indicators added yet.
                  </div>
                ) : (
                  <ScrollArea className="max-h-[280px] pr-3">
                    <div className="space-y-4">
                      {fields.map((field, index) => (
                        <div key={field.id} className="rounded-xl border bg-background p-4">
                          <div className="mb-3 flex items-center justify-between">
                            <h4 className="text-sm font-medium">Indicator {index + 1}</h4>
                            <Button type="button" variant="ghost" size="sm" onClick={() => remove(index)}>
                              Remove
                            </Button>
                          </div>
                          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                            <FormField name={`indicators.${index}.type`} label="Type" required>
                              <Select
                                value={methods.watch(`indicators.${index}.type`)}
                                onValueChange={(value) => methods.setValue(`indicators.${index}.type`, value as ThreatFormValues['indicators'][number]['type'])}
                              >
                                <SelectTrigger>
                                  <SelectValue />
                                </SelectTrigger>
                                <SelectContent>
                                  {INDICATOR_TYPE_OPTIONS.map((option) => (
                                    <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
                                  ))}
                                </SelectContent>
                              </Select>
                            </FormField>
                            <FormField name={`indicators.${index}.severity`} label="Severity" required>
                              <Select
                                value={methods.watch(`indicators.${index}.severity`)}
                                onValueChange={(value) => methods.setValue(`indicators.${index}.severity`, value as ThreatFormValues['indicators'][number]['severity'])}
                              >
                                <SelectTrigger>
                                  <SelectValue />
                                </SelectTrigger>
                                <SelectContent>
                                  {['critical', 'high', 'medium', 'low'].map((option) => (
                                    <SelectItem key={option} value={option} className="capitalize">{option}</SelectItem>
                                  ))}
                                </SelectContent>
                              </Select>
                            </FormField>
                            <FormField name={`indicators.${index}.value`} label="Value" required className="md:col-span-2">
                              <Input
                                placeholder="198.51.100.24 or auth-portal.example.com"
                                {...methods.register(`indicators.${index}.value`)}
                              />
                            </FormField>
                            <FormField name={`indicators.${index}.description`} label="Description" className="md:col-span-2">
                              <Textarea
                                rows={2}
                                placeholder="Observed in phishing callback infrastructure"
                                {...methods.register(`indicators.${index}.description`)}
                              />
                            </FormField>
                            <FormField name={`indicators.${index}.confidence`} label="Confidence">
                              <div className="rounded-xl border px-3 py-3">
                                <div className="mb-2 flex items-center justify-between text-xs text-muted-foreground">
                                  <span>Analyst confidence</span>
                                  <span>{Math.round(methods.watch(`indicators.${index}.confidence`) ?? 0)}%</span>
                                </div>
                                <Slider
                                  value={[methods.watch(`indicators.${index}.confidence`) ?? 0]}
                                  max={100}
                                  step={1}
                                  onValueChange={(value) => methods.setValue(`indicators.${index}.confidence`, value[0] ?? 0)}
                                />
                              </div>
                            </FormField>
                          </div>
                        </div>
                      ))}
                    </div>
                  </ScrollArea>
                )}
              </div>
            )}

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={isSubmitting}>
                {isSubmitting ? (isEditing ? 'Saving…' : 'Creating…') : (isEditing ? 'Save Changes' : 'Create Threat')}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}

function parseTags(value?: string): string[] | undefined {
  const tags = (value ?? '')
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);
  return tags.length > 0 ? Array.from(new Set(tags)) : undefined;
}
