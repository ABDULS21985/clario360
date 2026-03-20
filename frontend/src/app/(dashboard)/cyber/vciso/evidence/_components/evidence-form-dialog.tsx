'use client';

import { useState, useEffect } from 'react';
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
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import type { VCISOEvidence, EvidenceType, EvidenceSource } from '@/types/cyber';

interface EvidenceFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  evidence?: VCISOEvidence | null;
}

interface EvidenceFormData {
  title: string;
  description: string;
  type: EvidenceType;
  source: EvidenceSource;
  frameworks: string;
  control_ids: string;
  file_name: string;
  expires_at: string;
  collected_at: string;
}

const EVIDENCE_TYPES: { label: string; value: EvidenceType }[] = [
  { label: 'Screenshot', value: 'screenshot' },
  { label: 'Log', value: 'log' },
  { label: 'Configuration', value: 'config' },
  { label: 'Report', value: 'report' },
  { label: 'Policy', value: 'policy' },
  { label: 'Certificate', value: 'certificate' },
  { label: 'Other', value: 'other' },
];

const EVIDENCE_SOURCES: { label: string; value: EvidenceSource }[] = [
  { label: 'Manual', value: 'manual' },
  { label: 'Automated', value: 'automated' },
];

function getDefaultForm(evidence?: VCISOEvidence | null): EvidenceFormData {
  if (evidence) {
    return {
      title: evidence.title,
      description: evidence.description,
      type: evidence.type,
      source: evidence.source,
      frameworks: evidence.frameworks.join(', '),
      control_ids: evidence.control_ids.join(', '),
      file_name: evidence.file_name ?? '',
      expires_at: evidence.expires_at
        ? evidence.expires_at.slice(0, 10)
        : '',
      collected_at: evidence.collected_at
        ? evidence.collected_at.slice(0, 10)
        : '',
    };
  }
  return {
    title: '',
    description: '',
    type: 'report',
    source: 'manual',
    frameworks: '',
    control_ids: '',
    file_name: '',
    expires_at: '',
    collected_at: new Date().toISOString().slice(0, 10),
  };
}

function parseCommaSeparated(value: string): string[] {
  return value
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean);
}

export function EvidenceFormDialog({
  open,
  onOpenChange,
  evidence,
}: EvidenceFormDialogProps) {
  const isEdit = !!evidence;
  const [form, setForm] = useState<EvidenceFormData>(() => getDefaultForm(evidence));

  useEffect(() => {
    if (open) {
      setForm(getDefaultForm(evidence));
    }
  }, [open, evidence]);

  const { mutate: createEvidence, isPending: creating } = useApiMutation<
    VCISOEvidence,
    Record<string, unknown>
  >('post', API_ENDPOINTS.CYBER_VCISO_EVIDENCE, {
    successMessage: 'Evidence uploaded successfully',
    invalidateKeys: ['vciso-evidence', 'vciso-evidence-stats'],
    onSuccess: () => {
      onOpenChange(false);
    },
  });

  const { mutate: updateEvidence, isPending: updating } = useApiMutation<
    VCISOEvidence,
    Record<string, unknown>
  >(
    'put',
    () => `${API_ENDPOINTS.CYBER_VCISO_EVIDENCE}/${evidence?.id}`,
    {
      successMessage: 'Evidence updated successfully',
      invalidateKeys: ['vciso-evidence', 'vciso-evidence-stats'],
      onSuccess: () => {
        onOpenChange(false);
      },
    },
  );

  const isPending = creating || updating;

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();

    const payload: Record<string, unknown> = {
      title: form.title.trim(),
      description: form.description.trim(),
      type: form.type,
      source: form.source,
      frameworks: parseCommaSeparated(form.frameworks),
      control_ids: parseCommaSeparated(form.control_ids),
      collected_at: form.collected_at
        ? new Date(form.collected_at).toISOString()
        : new Date().toISOString(),
    };

    if (form.file_name.trim()) {
      payload.file_name = form.file_name.trim();
    }
    if (form.expires_at) {
      payload.expires_at = new Date(form.expires_at).toISOString();
    }

    if (isEdit) {
      updateEvidence(payload);
    } else {
      createEvidence(payload);
    }
  }

  function updateField<K extends keyof EvidenceFormData>(key: K, value: EvidenceFormData[K]) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  const isValid = form.title.trim().length > 0 && form.description.trim().length > 0;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{isEdit ? 'Edit Evidence' : 'Upload Evidence'}</DialogTitle>
          <DialogDescription>
            {isEdit
              ? 'Update the evidence details below.'
              : 'Add new evidence to the audit repository.'}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Title */}
          <div className="space-y-2">
            <Label htmlFor="evidence-title">Title *</Label>
            <Input
              id="evidence-title"
              placeholder="e.g., SOC 2 Access Control Screenshot"
              value={form.title}
              onChange={(e) => updateField('title', e.target.value)}
              required
            />
          </div>

          {/* Description */}
          <div className="space-y-2">
            <Label htmlFor="evidence-description">Description *</Label>
            <Textarea
              id="evidence-description"
              placeholder="Describe the evidence and its relevance..."
              value={form.description}
              onChange={(e) => updateField('description', e.target.value)}
              rows={3}
              required
            />
          </div>

          {/* Type & Source */}
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label>Type</Label>
              <Select
                value={form.type}
                onValueChange={(v) => updateField('type', v as EvidenceType)}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select type" />
                </SelectTrigger>
                <SelectContent>
                  {EVIDENCE_TYPES.map((t) => (
                    <SelectItem key={t.value} value={t.value}>
                      {t.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Source</Label>
              <Select
                value={form.source}
                onValueChange={(v) => updateField('source', v as EvidenceSource)}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select source" />
                </SelectTrigger>
                <SelectContent>
                  {EVIDENCE_SOURCES.map((s) => (
                    <SelectItem key={s.value} value={s.value}>
                      {s.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Frameworks */}
          <div className="space-y-2">
            <Label htmlFor="evidence-frameworks">Frameworks</Label>
            <Input
              id="evidence-frameworks"
              placeholder="SOC 2, ISO 27001, NIST CSF (comma-separated)"
              value={form.frameworks}
              onChange={(e) => updateField('frameworks', e.target.value)}
            />
            <p className="text-xs text-muted-foreground">Separate multiple frameworks with commas</p>
          </div>

          {/* Control IDs */}
          <div className="space-y-2">
            <Label htmlFor="evidence-controls">Control IDs</Label>
            <Input
              id="evidence-controls"
              placeholder="CC6.1, A.9.1.1, PR.AC-1 (comma-separated)"
              value={form.control_ids}
              onChange={(e) => updateField('control_ids', e.target.value)}
            />
            <p className="text-xs text-muted-foreground">Separate multiple control IDs with commas</p>
          </div>

          {/* File name */}
          <div className="space-y-2">
            <Label htmlFor="evidence-filename">File Name</Label>
            <Input
              id="evidence-filename"
              placeholder="access-review-2026-Q1.pdf"
              value={form.file_name}
              onChange={(e) => updateField('file_name', e.target.value)}
            />
          </div>

          {/* Dates */}
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="evidence-collected">Collected At</Label>
              <Input
                id="evidence-collected"
                type="date"
                value={form.collected_at}
                onChange={(e) => updateField('collected_at', e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="evidence-expires">Expires At</Label>
              <Input
                id="evidence-expires"
                type="date"
                value={form.expires_at}
                onChange={(e) => updateField('expires_at', e.target.value)}
              />
            </div>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isPending}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isPending || !isValid}>
              {isPending
                ? isEdit
                  ? 'Saving...'
                  : 'Uploading...'
                : isEdit
                  ? 'Save Changes'
                  : 'Upload Evidence'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
