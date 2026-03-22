'use client';

import { useEffect, useState } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
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
import { FormField } from '@/components/shared/forms/form-field';
import {
  enterpriseApi,
  lexDocumentSchema,
  type LexDocumentFormValues,
} from '@/lib/enterprise';
import { showApiError, showSuccess } from '@/lib/toast';
import type { FileRecord } from '@/types/models';
import type { LexDocument } from '@/types/suites';

interface DocumentFormDialogProps {
  document?: LexDocument | null;
  onOpenChange: (open: boolean) => void;
  onSaved?: (doc: LexDocument) => void;
  open: boolean;
}

const DOCUMENT_TYPES = [
  'policy',
  'regulation',
  'template',
  'memo',
  'opinion',
  'filing',
  'correspondence',
  'resolution',
  'power_of_attorney',
  'other',
] as const;

const CONFIDENTIALITY_LEVELS = [
  'public',
  'internal',
  'confidential',
  'privileged',
] as const;

const DOCUMENT_STATUSES = [
  { value: 'draft', label: 'Draft' },
  { value: 'active', label: 'Active' },
  { value: 'archived', label: 'Archived' },
  { value: 'superseded', label: 'Superseded' },
] as const;

export function DocumentFormDialog({
  document,
  onOpenChange,
  onSaved,
  open,
}: DocumentFormDialogProps) {
  const isEdit = Boolean(document);
  const queryClient = useQueryClient();
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [extractedText, setExtractedText] = useState('');
  const [changeSummary, setChangeSummary] = useState('');
  const [tagInputValue, setTagInputValue] = useState('');
  const [uploadProgress, setUploadProgress] = useState(0);

  const form = useForm<LexDocumentFormValues>({
    resolver: zodResolver(lexDocumentSchema),
    defaultValues: buildFormDefaults(document),
  });

  useEffect(() => {
    if (!open) return;
    form.reset(buildFormDefaults(document));
    setSelectedFile(null);
    setExtractedText('');
    setChangeSummary('');
    setTagInputValue(document?.tags.join(', ') ?? '');
    setUploadProgress(0);
  }, [document, form, open]);

  const saveMutation = useMutation({
    mutationFn: async (values: LexDocumentFormValues) => {
      let documentPayload: Record<string, unknown> | undefined;

      if (!isEdit && selectedFile) {
        const uploaded = await uploadDocumentFile(selectedFile, values, setUploadProgress);
        documentPayload = {
          file_id: uploaded.id,
          file_name: uploaded.original_name,
          file_size_bytes: uploaded.size_bytes,
          content_hash: uploaded.checksum_sha256,
          extracted_text: extractedText.trim(),
          change_summary: changeSummary.trim(),
        };
      }

      const payload = buildPayload(values, isEdit, documentPayload);
      if (isEdit && document) {
        return enterpriseApi.lex.updateDocument(document.id, payload);
      }
      return enterpriseApi.lex.createDocument(payload);
    },
    onSuccess: async (savedDoc) => {
      showSuccess(
        isEdit ? 'Document updated.' : 'Document created.',
        isEdit
          ? 'The document metadata has been saved.'
          : 'The legal document is now available in the repository.',
      );
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['lex-documents'] }),
        queryClient.invalidateQueries({ queryKey: ['lex-overview'] }),
        document
          ? queryClient.invalidateQueries({ queryKey: ['lex-document', document.id] })
          : Promise.resolve(),
      ]);
      onOpenChange(false);
      onSaved?.(savedDoc);
    },
    onError: showApiError,
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] max-w-2xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{isEdit ? 'Edit Document' : 'Create Document'}</DialogTitle>
          <DialogDescription>
            {isEdit
              ? 'Update legal document metadata and classification.'
              : 'Register a new legal document and optionally attach the initial file.'}
          </DialogDescription>
        </DialogHeader>

        <FormProvider {...form}>
          <form
            className="space-y-5"
            onSubmit={form.handleSubmit((values) => saveMutation.mutate(values))}
          >
            <FormField name="title" label="Title" required>
              <Input id="title" {...form.register('title')} placeholder="Data Protection Policy" />
            </FormField>

            {isEdit ? (
              <FormField name="status" label="Status" required>
                <Select
                  value={form.watch('status') ?? 'active'}
                  onValueChange={(value) =>
                    form.setValue('status', value as NonNullable<LexDocumentFormValues['status']>, { shouldValidate: true })
                  }
                >
                  <SelectTrigger id="status">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {DOCUMENT_STATUSES.map((s) => (
                      <SelectItem key={s.value} value={s.value}>
                        {s.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
            ) : null}

            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <FormField name="type" label="Document type" required>
                <Select
                  value={form.watch('type')}
                  onValueChange={(value) =>
                    form.setValue('type', value as LexDocumentFormValues['type'], { shouldValidate: true })
                  }
                >
                  <SelectTrigger id="type">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {DOCUMENT_TYPES.map((type) => (
                      <SelectItem key={type} value={type}>
                        {type.replace(/_/g, ' ')}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>

              <FormField name="confidentiality" label="Confidentiality" required>
                <Select
                  value={form.watch('confidentiality')}
                  onValueChange={(value) =>
                    form.setValue('confidentiality', value as LexDocumentFormValues['confidentiality'], { shouldValidate: true })
                  }
                >
                  <SelectTrigger id="confidentiality">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {CONFIDENTIALITY_LEVELS.map((level) => (
                      <SelectItem key={level} value={level}>
                        {level.charAt(0).toUpperCase() + level.slice(1)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
            </div>

            <FormField name="description" label="Description" required>
              <Textarea
                id="description"
                {...form.register('description')}
                placeholder="Scope, purpose, and applicability of this legal document."
                rows={3}
              />
            </FormField>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <FormField name="category" label="Category">
                <Input id="category" {...form.register('category')} placeholder="Compliance" />
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
                  placeholder="gdpr, policy, internal"
                />
              </FormField>
            </div>

            {!isEdit ? (
              <div className="space-y-4 rounded-lg border px-4 py-4">
                <div>
                  <p className="text-sm font-medium">Initial document file</p>
                  <p className="text-xs text-muted-foreground">
                    Optional. Upload the source file to enable version tracking from the start.
                  </p>
                </div>

                <FormField name="initial_file" label="Document file">
                  <Input
                    id="initial_file"
                    type="file"
                    accept=".pdf,.doc,.docx,.txt,.xlsx,.pptx"
                    onChange={(event) => setSelectedFile(event.target.files?.[0] ?? null)}
                  />
                </FormField>

                {selectedFile ? (
                  <p className="text-xs text-muted-foreground">
                    Selected: {selectedFile.name}
                  </p>
                ) : null}

                <FormField name="document_extracted_text" label="Extracted text">
                  <Textarea
                    id="document_extracted_text"
                    value={extractedText}
                    onChange={(event) => setExtractedText(event.target.value)}
                    placeholder="Paste document text for indexing."
                    rows={4}
                  />
                </FormField>

                <FormField name="document_change_summary" label="Change summary">
                  <Input
                    id="document_change_summary"
                    value={changeSummary}
                    onChange={(event) => setChangeSummary(event.target.value)}
                    placeholder="Initial published version"
                  />
                </FormField>

                {saveMutation.isPending && selectedFile ? (
                  <p className="text-xs text-muted-foreground">
                    Upload progress: {Math.round(uploadProgress)}%
                  </p>
                ) : null}
              </div>
            ) : null}

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={saveMutation.isPending}>
                {saveMutation.isPending ? (
                  <Loader2 className="mr-1.5 h-4 w-4 animate-spin" />
                ) : null}
                {isEdit ? 'Save changes' : 'Create document'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}

function buildFormDefaults(doc?: LexDocument | null): LexDocumentFormValues {
  return {
    title: doc?.title ?? '',
    type: doc?.type ?? 'policy',
    description: doc?.description ?? '',
    category: doc?.category ?? null,
    confidentiality: doc?.confidentiality ?? 'internal',
    status: doc?.status ?? 'active',
    contract_id: doc?.contract_id ?? null,
    tags: doc?.tags ?? [],
    metadata: doc?.metadata ?? {},
    document: null,
  };
}

function buildPayload(
  values: LexDocumentFormValues,
  isEdit: boolean,
  documentPayload?: Record<string, unknown>,
): Record<string, unknown> {
  if (isEdit) {
    return {
      title: values.title.trim(),
      type: values.type,
      description: values.description.trim(),
      category: values.category?.trim() ?? '',
      confidentiality: values.confidentiality,
      status: values.status ?? 'active',
      contract_id: values.contract_id ?? null,
      tags: values.tags,
      metadata: values.metadata ?? {},
    };
  }
  return {
    title: values.title.trim(),
    type: values.type,
    description: values.description.trim(),
    category: values.category?.trim() || null,
    confidentiality: values.confidentiality,
    contract_id: values.contract_id ?? null,
    tags: values.tags,
    metadata: values.metadata ?? {},
    ...(documentPayload ? { document: documentPayload } : {}),
  };
}

async function uploadDocumentFile(
  file: File,
  values: LexDocumentFormValues,
  onProgress: (progress: number) => void,
): Promise<FileRecord> {
  return enterpriseApi.files.upload(
    file,
    {
      suite: 'lex',
      entity_type: 'document',
      tags: Array.from(new Set(['document', values.type, ...values.tags])).join(','),
      lifecycle_policy: 'standard',
    },
    onProgress,
  );
}

function parseTagInput(value: string): string[] {
  return value
    .split(',')
    .map((item) => item.trim().toLowerCase())
    .filter(Boolean);
}
