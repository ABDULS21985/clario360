'use client';

import { useEffect } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useForm } from 'react-hook-form';
import { z } from 'zod';
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
import { FormField } from '@/components/shared/forms/form-field';
import {
  buildTagInputValue,
  CTI_ACTOR_MOTIVATION_OPTIONS,
  CTI_SOPHISTICATION_OPTIONS,
  CTI_THREAT_ACTOR_OPTIONS,
  countryCodeToFlag,
  parseTagInput,
} from '@/lib/cti-utils';
import type { CreateThreatActorRequest, CTIThreatActor, UpdateThreatActorRequest } from '@/types/cti';

const actorSchema = z.object({
  name: z.string().min(1, 'Name is required').max(300),
  aliases_input: z.string().optional(),
  actor_type: z.enum(['state_sponsored', 'cybercriminal', 'hacktivist', 'insider', 'unknown']),
  origin_country_code: z.string().max(2, 'Use an ISO country code').optional(),
  sophistication_level: z.enum(['advanced', 'intermediate', 'basic']),
  primary_motivation: z.enum(['espionage', 'financial_gain', 'disruption', 'ideological', 'unknown']),
  description: z.string().optional(),
  mitre_group_id: z.string().optional(),
  risk_score: z.coerce.number().min(0).max(100),
});

const COUNTRY_OPTIONS = [
  { code: 'CN', label: 'China' },
  { code: 'RU', label: 'Russia' },
  { code: 'IR', label: 'Iran' },
  { code: 'KP', label: 'North Korea' },
  { code: 'US', label: 'United States' },
  { code: 'NG', label: 'Nigeria' },
  { code: 'GB', label: 'United Kingdom' },
  { code: 'DE', label: 'Germany' },
  { code: 'BR', label: 'Brazil' },
  { code: 'IN', label: 'India' },
];

type ActorFormValues = z.infer<typeof actorSchema>;

interface ActorFormProps {
  actor?: CTIThreatActor | null;
  onSubmit: (payload: CreateThreatActorRequest | UpdateThreatActorRequest) => Promise<void>;
  onCancel: () => void;
  isLoading?: boolean;
}

export function ActorForm({
  actor,
  onSubmit,
  onCancel,
  isLoading = false,
}: ActorFormProps) {
  const form = useForm<ActorFormValues>({
    resolver: zodResolver(actorSchema),
    defaultValues: {
      name: '',
      aliases_input: '',
      actor_type: 'unknown',
      origin_country_code: '',
      sophistication_level: 'intermediate',
      primary_motivation: 'unknown',
      description: '',
      mitre_group_id: '',
      risk_score: 50,
    },
  });

  useEffect(() => {
    form.reset({
      name: actor?.name ?? '',
      aliases_input: buildTagInputValue(actor?.aliases),
      actor_type: actor?.actor_type ?? 'unknown',
      origin_country_code: actor?.origin_country_code ?? '',
      sophistication_level: actor?.sophistication_level ?? 'intermediate',
      primary_motivation: actor?.primary_motivation ?? 'unknown',
      description: actor?.description ?? '',
      mitre_group_id: actor?.mitre_group_id ?? '',
      risk_score: actor?.risk_score ?? 50,
    });
  }, [actor, form]);

  const handleSubmit = form.handleSubmit(async (values) => {
    const payload = {
      name: values.name.trim(),
      aliases: parseTagInput(values.aliases_input ?? ''),
      actor_type: values.actor_type,
      origin_country_code: values.origin_country_code?.trim().toUpperCase() || undefined,
      sophistication_level: values.sophistication_level,
      primary_motivation: values.primary_motivation,
      description: values.description?.trim() || undefined,
      mitre_group_id: values.mitre_group_id?.trim() || undefined,
      risk_score: values.risk_score,
    };

    await onSubmit(payload);
  });

  return (
    <FormProvider {...form}>
      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="grid gap-4 md:grid-cols-2">
          <FormField name="name" label="Actor Name" required>
            <Input id="name" placeholder="APT29" {...form.register('name')} />
          </FormField>
          <FormField name="actor_type" label="Actor Type" required>
            <Select
              value={form.watch('actor_type')}
              onValueChange={(value) => form.setValue('actor_type', value as ActorFormValues['actor_type'])}
            >
              <SelectTrigger id="actor_type">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {CTI_THREAT_ACTOR_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FormField>
          <FormField name="aliases_input" label="Aliases">
            <Input id="aliases_input" placeholder="Cozy Bear, Midnight Blizzard" {...form.register('aliases_input')} />
          </FormField>
          <FormField name="origin_country_code" label="Origin Country">
            <Select
              value={(form.watch('origin_country_code') || '').toUpperCase() || 'none'}
              onValueChange={(value) => form.setValue('origin_country_code', value === 'none' ? '' : value)}
            >
              <SelectTrigger id="origin_country_code">
                <SelectValue placeholder="Select origin country" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">Unknown</SelectItem>
                {COUNTRY_OPTIONS.map((country) => (
                  <SelectItem key={country.code} value={country.code}>
                    {countryCodeToFlag(country.code)} {country.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FormField>
          <FormField name="sophistication_level" label="Sophistication" required>
            <Select
              value={form.watch('sophistication_level')}
              onValueChange={(value) => form.setValue('sophistication_level', value as ActorFormValues['sophistication_level'])}
            >
              <SelectTrigger id="sophistication_level">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {CTI_SOPHISTICATION_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FormField>
          <FormField name="primary_motivation" label="Primary Motivation" required>
            <Select
              value={form.watch('primary_motivation')}
              onValueChange={(value) => form.setValue('primary_motivation', value as ActorFormValues['primary_motivation'])}
            >
              <SelectTrigger id="primary_motivation">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {CTI_ACTOR_MOTIVATION_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FormField>
          <FormField name="mitre_group_id" label="MITRE Group ID">
            <Input id="mitre_group_id" placeholder="G0016" {...form.register('mitre_group_id')} />
          </FormField>
          <FormField name="risk_score" label="Risk Score" required>
            <Input id="risk_score" type="number" min={0} max={100} step={1} {...form.register('risk_score')} />
          </FormField>
        </div>
        <FormField name="description" label="Description">
          <Textarea
            id="description"
            rows={5}
            placeholder="Operational notes, known tooling, targeting patterns, and confidence caveats."
            {...form.register('description')}
          />
        </FormField>
        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" onClick={onCancel}>
            Cancel
          </Button>
          <Button type="submit" disabled={isLoading}>
            {isLoading ? 'Saving...' : actor ? 'Save Changes' : 'Create Actor'}
          </Button>
        </div>
      </form>
    </FormProvider>
  );
}
