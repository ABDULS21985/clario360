'use client';

import { useEffect, useMemo, useState } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { FormProvider, useForm } from 'react-hook-form';
import { Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { FormField } from '@/components/shared/forms/form-field';
import { useAuth } from '@/hooks/use-auth';
import {
  enterpriseApi,
  lexContractSchema,
  type LexContractFormValues,
  userDisplayName,
} from '@/lib/enterprise';
import { showApiError, showSuccess } from '@/lib/toast';
import type { FileUploadRecord, LexContractRecord, UserDirectoryEntry } from '@/types/suites';

interface ContractFormDialogProps {
  contract?: LexContractRecord | null;
  onOpenChange: (open: boolean) => void;
  onSaved?: (contract: LexContractRecord) => void;
  open: boolean;
}

const CONTRACT_TYPES = [
  'service_agreement',
  'nda',
  'employment',
  'vendor',
  'license',
  'lease',
  'partnership',
  'consulting',
  'procurement',
  'sla',
  'mou',
  'amendment',
  'renewal',
  'other',
] as const;

export function ContractFormDialog({
  contract,
  onOpenChange,
  onSaved,
  open,
}: ContractFormDialogProps) {
  const isEdit = Boolean(contract);
  const queryClient = useQueryClient();
  const { user } = useAuth();
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [documentExtractedText, setDocumentExtractedText] = useState('');
  const [documentChangeSummary, setDocumentChangeSummary] = useState('');
  const [tagInputValue, setTagInputValue] = useState('');
  const [uploadProgress, setUploadProgress] = useState(0);

  const usersQuery = useQuery({
    queryKey: ['enterprise-users', 'lex-contract-dialog'],
    queryFn: () => enterpriseApi.users.list({ page: 1, per_page: 200, order: 'asc' }),
    enabled: open,
  });

  const form = useForm<LexContractFormValues>({
    resolver: zodResolver(lexContractSchema),
    defaultValues: buildFormDefaults(contract),
  });

  const users = usersQuery.data?.data ?? [];
  const usersById = useMemo(
    () => new Map(users.map((user) => [user.id, user])),
    [users],
  );

  useEffect(() => {
    if (!open) {
      return;
    }
    form.reset(buildFormDefaults(contract));
    setSelectedFile(null);
    setDocumentExtractedText('');
    setDocumentChangeSummary('');
    setTagInputValue(buildTagInputValue(contract?.tags));
    setUploadProgress(0);
  }, [contract, form, open]);

  useEffect(() => {
    if (!open || isEdit || users.length === 0) {
      return;
    }

    const currentOwnerId = form.getValues('owner_user_id');
    if (currentOwnerId) {
      return;
    }

    const currentUserId = user?.id;
    if (!currentUserId || !usersById.has(currentUserId)) {
      return;
    }

    applyOwner(currentUserId);
  }, [form, isEdit, open, user?.id, users.length, usersById]);

  const applyOwner = (userId: string) => {
    const user = usersById.get(userId);
    form.setValue('owner_user_id', userId, { shouldValidate: true });
    form.setValue('owner_name', user ? userDisplayName(user) : '', { shouldValidate: true });
  };

  const applyReviewer = (userId: string | null) => {
    const user = userId ? usersById.get(userId) : undefined;
    form.setValue('legal_reviewer_id', userId, { shouldValidate: true });
    form.setValue('legal_reviewer_name', user ? userDisplayName(user) : '', { shouldValidate: true });
  };

  const saveMutation = useMutation({
    mutationFn: async (values: LexContractFormValues) => {
      let documentPayload: Record<string, unknown> | undefined;

      if (!isEdit && selectedFile) {
        const uploaded = await uploadContractDocument({
          changeSummary: documentChangeSummary,
          extractedText: documentExtractedText,
          file: selectedFile,
          onProgress: setUploadProgress,
          type: values.type,
          tags: values.tags,
        });
        documentPayload = toFileReference(uploaded, documentExtractedText, documentChangeSummary);
      }

      const payload = buildContractPayload(values, documentPayload);
      if (isEdit && contract) {
        return enterpriseApi.lex.updateContract(contract.id, payload);
      }
      return enterpriseApi.lex.createContract(payload);
    },
    onSuccess: async (savedContract) => {
      showSuccess(
        isEdit ? 'Contract updated.' : 'Contract created.',
        isEdit
          ? 'The contract metadata has been saved.'
          : 'The contract record is now available in Clario Lex.',
      );
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['lex-contracts'] }),
        queryClient.invalidateQueries({ queryKey: ['lex-overview'] }),
        contract
          ? queryClient.invalidateQueries({ queryKey: ['lex-contract', contract.id] })
          : Promise.resolve(),
      ]);
      onOpenChange(false);
      onSaved?.(savedContract);
    },
    onError: showApiError,
  });

  const ownerValue = form.watch('owner_user_id');
  const reviewerValue = form.watch('legal_reviewer_id');

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] max-w-4xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{isEdit ? 'Edit Contract' : 'Create Contract'}</DialogTitle>
          <DialogDescription>
            {isEdit
              ? 'Update contract metadata, ownership, dates, and lifecycle context.'
              : 'Register a new contract and optionally attach the first document version.'}
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...form}>
          <form
            className="space-y-5"
            onSubmit={form.handleSubmit((values) => saveMutation.mutate(values))}
          >
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <FormField name="title" label="Title" required>
                <Input id="title" {...form.register('title')} placeholder="Master Services Agreement" />
              </FormField>

              <FormField name="contract_number" label="Contract number">
                <Input id="contract_number" {...form.register('contract_number')} placeholder="LEX-2026-001" />
              </FormField>
            </div>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <FormField name="type" label="Contract type" required>
                <Select
                  value={form.watch('type')}
                  onValueChange={(value) =>
                    form.setValue('type', value as LexContractFormValues['type'], { shouldValidate: true })
                  }
                >
                  <SelectTrigger id="type">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {CONTRACT_TYPES.map((type) => (
                      <SelectItem key={type} value={type}>
                        {type.replace(/_/g, ' ')}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>

              <FormField name="currency" label="Currency" required>
                <Input
                  id="currency"
                  {...form.register('currency')}
                  maxLength={3}
                  placeholder="USD"
                  onChange={(event) =>
                    form.setValue('currency', event.target.value.toUpperCase(), { shouldValidate: true })
                  }
                />
              </FormField>
            </div>

            <FormField name="description" label="Description" required>
              <Textarea
                id="description"
                {...form.register('description')}
                placeholder="Commercial scope, renewal expectations, service obligations, and key legal posture."
                rows={3}
              />
            </FormField>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <FormField name="party_a_name" label="Party A" required>
                <Input id="party_a_name" {...form.register('party_a_name')} placeholder="Clario 360 Ltd." />
              </FormField>

              <FormField name="party_b_name" label="Counterparty" required>
                <Input id="party_b_name" {...form.register('party_b_name')} placeholder="Acme Holdings" />
              </FormField>
            </div>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
              <FormField name="party_a_entity" label="Party A entity">
                <Input id="party_a_entity" {...form.register('party_a_entity')} placeholder="Legal entity name" />
              </FormField>

              <FormField name="party_b_entity" label="Counterparty entity">
                <Input id="party_b_entity" {...form.register('party_b_entity')} placeholder="Legal entity name" />
              </FormField>

              <FormField name="party_b_contact" label="Counterparty contact">
                <Input id="party_b_contact" {...form.register('party_b_contact')} placeholder="legal@acme.example" />
              </FormField>
            </div>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <FormField name="owner_user_id" label="Contract owner" required>
                <Select value={ownerValue} onValueChange={applyOwner}>
                  <SelectTrigger id="owner_user_id">
                    <SelectValue placeholder="Select owner" />
                  </SelectTrigger>
                  <SelectContent>
                    {users.map((user) => (
                      <SelectItem key={user.id} value={user.id}>
                        {userDisplayName(user)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>

              <FormField name="legal_reviewer_id" label="Legal reviewer">
                <Select
                  value={reviewerValue ?? 'none'}
                  onValueChange={(value) => applyReviewer(value === 'none' ? null : value)}
                >
                  <SelectTrigger id="legal_reviewer_id">
                    <SelectValue placeholder="Select reviewer" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="none">Unassigned</SelectItem>
                    {users.map((user) => (
                      <SelectItem key={user.id} value={user.id}>
                        {userDisplayName(user)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
            </div>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
              <FormField name="total_value" label="Total value">
                <Input
                  id="total_value"
                  type="number"
                  inputMode="decimal"
                  min={0}
                  step="0.01"
                  {...form.register('total_value', {
                    setValueAs: (value) => (value === '' ? null : Number(value)),
                  })}
                  placeholder="125000"
                />
              </FormField>

              <FormField name="effective_date" label="Effective date">
                <Input id="effective_date" type="date" {...form.register('effective_date')} />
              </FormField>

              <FormField name="expiry_date" label="Expiry date">
                <Input id="expiry_date" type="date" {...form.register('expiry_date')} />
              </FormField>
            </div>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
              <FormField name="renewal_date" label="Renewal date">
                <Input id="renewal_date" type="date" {...form.register('renewal_date')} />
              </FormField>

              <FormField name="renewal_notice_days" label="Renewal notice (days)" required>
                <Input
                  id="renewal_notice_days"
                  type="number"
                  min={0}
                  max={365}
                  {...form.register('renewal_notice_days', {
                    setValueAs: (value) => Number(value),
                  })}
                />
              </FormField>

              <FormField name="department" label="Department">
                <Input id="department" {...form.register('department')} placeholder="Procurement" />
              </FormField>
            </div>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <FormField name="payment_terms" label="Payment terms">
                <Input id="payment_terms" {...form.register('payment_terms')} placeholder="Net 30" />
              </FormField>

              <FormField name="tags" label="Tags">
                <Input
                  id="tags"
                  value={tagInputValue}
                  onChange={(event) => {
                    const nextValue = event.target.value;
                    setTagInputValue(nextValue);
                    form.setValue('tags', parseTagInput(nextValue), { shouldValidate: true });
                  }}
                  placeholder="msa, vendor, renewal"
                />
              </FormField>
            </div>

            <div className="rounded-lg border px-4 py-4">
              <div className="flex items-center justify-between gap-4">
                <div>
                  <p className="text-sm font-medium">Auto-renew</p>
                  <p className="text-xs text-muted-foreground">
                    Mark whether the contract renews automatically unless terminated.
                  </p>
                </div>
                <Switch
                  checked={form.watch('auto_renew')}
                  onCheckedChange={(checked) =>
                    form.setValue('auto_renew', checked, { shouldValidate: true })
                  }
                />
              </div>
            </div>

            {!isEdit ? (
              <div className="space-y-4 rounded-lg border px-4 py-4">
                <div>
                  <p className="text-sm font-medium">Initial document version</p>
                  <p className="text-xs text-muted-foreground">
                    Optional. Uploading the first version now enables immediate analysis and clause extraction.
                  </p>
                </div>

                <FormField name="initial_document" label="Contract file">
                  <Input
                    id="initial_document"
                    type="file"
                    accept=".pdf,.doc,.docx,.txt"
                    onChange={(event) => setSelectedFile(event.target.files?.[0] ?? null)}
                  />
                </FormField>

                {selectedFile ? (
                  <p className="text-xs text-muted-foreground">
                    Selected: {selectedFile.name}
                  </p>
                ) : null}

                <FormField name="document_extracted_text" label="Document text">
                  <Textarea
                    id="document_extracted_text"
                    value={documentExtractedText}
                    onChange={(event) => setDocumentExtractedText(event.target.value)}
                    placeholder="Paste contract text to enable deterministic analysis immediately after upload."
                    rows={5}
                  />
                </FormField>

                <FormField name="document_change_summary" label="Change summary">
                  <Input
                    id="document_change_summary"
                    value={documentChangeSummary}
                    onChange={(event) => setDocumentChangeSummary(event.target.value)}
                    placeholder="Initial signed draft"
                  />
                </FormField>

                {saveMutation.isPending && selectedFile ? (
                  <p className="text-xs text-muted-foreground">
                    Upload progress: {Math.round(uploadProgress)}%
                  </p>
                ) : null}
              </div>
            ) : null}

            {usersQuery.isError ? (
              <p className="text-sm text-destructive">
                Unable to load the user directory. Contract save is disabled until owners can be resolved.
              </p>
            ) : null}

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button
                type="submit"
                disabled={saveMutation.isPending || usersQuery.isLoading || usersQuery.isError}
              >
                {saveMutation.isPending ? (
                  <Loader2 className="mr-1.5 h-4 w-4 animate-spin" />
                ) : null}
                {isEdit ? 'Save changes' : 'Create contract'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}

function buildFormDefaults(contract?: LexContractRecord | null): LexContractFormValues {
  return {
    title: contract?.title ?? '',
    contract_number: contract?.contract_number ?? null,
    type: contract?.type ?? 'service_agreement',
    description: contract?.description ?? '',
    party_a_name: contract?.party_a_name ?? '',
    party_a_entity: contract?.party_a_entity ?? null,
    party_b_name: contract?.party_b_name ?? '',
    party_b_entity: contract?.party_b_entity ?? null,
    party_b_contact: contract?.party_b_contact ?? null,
    total_value: contract?.total_value ?? null,
    currency: contract?.currency ?? 'SAR',
    payment_terms: contract?.payment_terms ?? null,
    effective_date: toDateInputValue(contract?.effective_date),
    expiry_date: toDateInputValue(contract?.expiry_date),
    renewal_date: toDateInputValue(contract?.renewal_date),
    auto_renew: contract?.auto_renew ?? false,
    renewal_notice_days: contract?.renewal_notice_days ?? 30,
    owner_user_id: contract?.owner_user_id ?? '',
    owner_name: contract?.owner_name ?? '',
    legal_reviewer_id: contract?.legal_reviewer_id ?? null,
    legal_reviewer_name: contract?.legal_reviewer_name ?? null,
    department: contract?.department ?? null,
    tags: contract?.tags ?? [],
    metadata: contract?.metadata ?? {},
    document: null,
  };
}

function buildContractPayload(
  values: LexContractFormValues,
  documentPayload?: Record<string, unknown>,
): Record<string, unknown> {
  return {
    title: values.title.trim(),
    contract_number: emptyToNull(values.contract_number),
    type: values.type,
    description: values.description.trim(),
    party_a_name: values.party_a_name.trim(),
    party_a_entity: emptyToNull(values.party_a_entity),
    party_b_name: values.party_b_name.trim(),
    party_b_entity: emptyToNull(values.party_b_entity),
    party_b_contact: emptyToNull(values.party_b_contact),
    total_value: values.total_value ?? null,
    currency: values.currency.trim().toUpperCase(),
    payment_terms: emptyToNull(values.payment_terms),
    effective_date: toOptionalDateTime(values.effective_date),
    expiry_date: toOptionalDateTime(values.expiry_date),
    renewal_date: toOptionalDateTime(values.renewal_date),
    auto_renew: values.auto_renew,
    renewal_notice_days: values.renewal_notice_days,
    owner_user_id: values.owner_user_id,
    owner_name: values.owner_name.trim(),
    legal_reviewer_id: values.legal_reviewer_id ?? null,
    legal_reviewer_name: emptyToNull(values.legal_reviewer_name),
    department: emptyToNull(values.department),
    tags: values.tags,
    metadata: values.metadata ?? {},
    ...(documentPayload ? { document: documentPayload } : {}),
  };
}

async function uploadContractDocument({
  changeSummary,
  extractedText,
  file,
  onProgress,
  tags,
  type,
}: {
  changeSummary: string;
  extractedText: string;
  file: File;
  onProgress: (progress: number) => void;
  tags: string[];
  type: LexContractFormValues['type'];
}): Promise<FileUploadRecord> {
  return enterpriseApi.files.upload(
    file,
    {
      suite: 'lex',
      entity_type: 'contract',
      tags: Array.from(new Set(['contract', type, ...tags])).join(','),
      lifecycle_policy: 'standard',
    },
    onProgress,
  );
}

function toFileReference(
  upload: FileUploadRecord,
  extractedText: string,
  changeSummary: string,
): Record<string, unknown> {
  return {
    file_id: upload.id,
    file_name: upload.original_name,
    file_size_bytes: upload.size_bytes,
    content_hash: upload.checksum_sha256,
    extracted_text: extractedText.trim(),
    change_summary: changeSummary.trim(),
  };
}

function parseTagInput(value: string): string[] {
  return value
    .split(',')
    .map((item) => item.trim().toLowerCase())
    .filter(Boolean);
}

function buildTagInputValue(tags?: string[] | null): string {
  return tags?.join(', ') ?? '';
}

function toDateInputValue(value?: string | null): string {
  if (!value) {
    return '';
  }
  return new Date(value).toISOString().slice(0, 10);
}

function toOptionalDateTime(value?: string | null): string | null {
  if (!value) {
    return null;
  }
  return new Date(`${value}T00:00:00Z`).toISOString();
}

function emptyToNull(value?: string | null): string | null {
  const trimmed = value?.trim();
  return trimmed ? trimmed : null;
}
