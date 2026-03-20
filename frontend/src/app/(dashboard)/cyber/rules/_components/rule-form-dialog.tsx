'use client';

import { useEffect, useState } from 'react';
import { useForm, FormProvider } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { ChevronDown, ChevronRight } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { ScrollArea } from '@/components/ui/scroll-area';
import { FormField } from '@/components/shared/forms/form-field';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import { normalizeRuleContent } from '@/lib/cyber-rules';
import type {
  DetectionRule,
  SigmaRuleContent,
  ThresholdRuleContent,
  AnomalyRuleContent,
  CorrelationRuleContent,
} from '@/types/cyber';

import { RuleSigmaEditor, defaultSigmaContent } from './rule-sigma-editor';
import { RuleThresholdEditor, defaultThresholdContent } from './rule-threshold-editor';
import { RuleAnomalyEditor, defaultAnomalyContent } from './rule-anomaly-editor';
import { RuleCorrelationEditor, defaultCorrelationContent } from './rule-correlation-editor';
import { RuleMitreSelector } from './rule-mitre-selector';

const schema = z.object({
  name: z.string().min(2).max(255),
  description: z.string().optional().or(z.literal('')),
  type: z.enum(['sigma', 'threshold', 'correlation', 'anomaly']),
  severity: z.enum(['critical', 'high', 'medium', 'low', 'info']),
  base_confidence: z.number().min(0).max(1).default(0.7),
});

type FormValues = z.infer<typeof schema>;

type RuleType = 'sigma' | 'threshold' | 'correlation' | 'anomaly';

interface RuleFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  rule?: DetectionRule | null;
  initialTechniqueId?: string | null;
  onSuccess?: () => void;
}

function getDefaultContent(
  type: RuleType,
  rule?: DetectionRule | null,
): SigmaRuleContent | ThresholdRuleContent | AnomalyRuleContent | CorrelationRuleContent {
  if (rule?.rule_content) {
    return normalizeRuleContent(type, rule.rule_content) as
      | SigmaRuleContent
      | ThresholdRuleContent
      | AnomalyRuleContent
      | CorrelationRuleContent;
  }
  if (type === 'threshold') return defaultThresholdContent();
  if (type === 'anomaly') return defaultAnomalyContent();
  if (type === 'correlation') return defaultCorrelationContent();
  return defaultSigmaContent();
}

export function RuleFormDialog({
  open,
  onOpenChange,
  rule,
  initialTechniqueId,
  onSuccess,
}: RuleFormDialogProps) {
  const isEdit = !!rule;

  const methods = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: '',
      description: '',
      type: 'sigma',
      severity: 'medium',
      base_confidence: 0.7,
    },
  });

  const selectedType = methods.watch('type') as RuleType;

  const [ruleContent, setRuleContent] = useState<
    SigmaRuleContent | ThresholdRuleContent | AnomalyRuleContent | CorrelationRuleContent
  >(defaultSigmaContent());
  const [mitreIds, setMitreIds] = useState<string[]>([]);
  const [previewOpen, setPreviewOpen] = useState(false);

  useEffect(() => {
    if (open) {
      methods.reset({
        name: rule?.name ?? '',
        description: rule?.description ?? '',
        type: (rule?.type ?? 'sigma') as RuleType,
        severity: rule?.severity ?? 'medium',
        base_confidence: rule?.base_confidence ?? 0.7,
      });
      setMitreIds(
        rule?.mitre_technique_ids ??
          (initialTechniqueId ? [initialTechniqueId] : []),
      );
      setRuleContent(getDefaultContent((rule?.type ?? 'sigma') as RuleType, rule));
      setPreviewOpen(false);
    }
  }, [open, rule, initialTechniqueId, methods]);

  // When type changes, reset rule content to defaults
  useEffect(() => {
    if (!rule?.rule_content) {
      setRuleContent(getDefaultContent(selectedType));
    }
  }, [selectedType, rule?.rule_content]);

  const url = isEdit
    ? `${API_ENDPOINTS.CYBER_RULES}/${rule!.id}`
    : API_ENDPOINTS.CYBER_RULES;

  const { mutate, isPending } = useApiMutation<DetectionRule, unknown>(
    isEdit ? 'put' : 'post',
    url,
    {
      successMessage: isEdit ? 'Rule updated' : 'Rule created',
      invalidateKeys: ['cyber-rules'],
      onSuccess: () => {
        onOpenChange(false);
        onSuccess?.();
      },
    },
  );

  const onSubmit = methods.handleSubmit((data) => {
    const payload = {
      ...data,
      rule_content: ruleContent,
      mitre_technique_ids: mitreIds,
    };
    mutate(payload);
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[90vh] max-w-2xl flex-col p-0">
        <DialogHeader className="border-b px-6 py-4">
          <DialogTitle>{isEdit ? 'Edit Detection Rule' : 'Create Detection Rule'}</DialogTitle>
        </DialogHeader>

        <ScrollArea className="flex-1">
          <FormProvider {...methods}>
            <form id="rule-form" onSubmit={onSubmit} className="space-y-5 px-6 py-4">
              {/* Common fields */}
              <FormField name="name" label="Rule Name" required>
                <Input placeholder="Suspicious PowerShell Execution" {...methods.register('name')} />
              </FormField>

              <FormField name="description" label="Description">
                <Textarea
                  rows={2}
                  placeholder="Describe what this rule detects…"
                  {...methods.register('description')}
                />
              </FormField>

              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div>
                  <Label className="text-sm font-medium">Rule Type</Label>
                  <RadioGroup
                    value={selectedType}
                    onValueChange={(v) => methods.setValue('type', v as RuleType)}
                    className="mt-2 grid grid-cols-1 gap-2 sm:grid-cols-2"
                  >
                    {(['sigma', 'threshold', 'correlation', 'anomaly'] as const).map((t) => (
                      <div key={t} className="flex items-center gap-2">
                        <RadioGroupItem value={t} id={`type-${t}`} />
                        <Label htmlFor={`type-${t}`} className="cursor-pointer capitalize text-sm">
                          {t}
                        </Label>
                      </div>
                    ))}
                  </RadioGroup>
                </div>

                <div className="space-y-3">
                  <FormField name="severity" label="Severity" required>
                    <Select
                      value={methods.watch('severity')}
                      onValueChange={(v) => methods.setValue('severity', v as FormValues['severity'])}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {(['critical', 'high', 'medium', 'low', 'info'] as const).map((s) => (
                          <SelectItem key={s} value={s} className="capitalize">{s}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </FormField>

                  <FormField name="base_confidence" label="Base Confidence (0–1)">
                    <Input
                      type="number"
                      step={0.05}
                      min={0}
                      max={1}
                      {...methods.register('base_confidence', { valueAsNumber: true })}
                    />
                  </FormField>
                </div>
              </div>

              {/* MITRE Techniques */}
              <div>
                <Label className="mb-1.5 block text-sm font-medium">MITRE Techniques</Label>
                <RuleMitreSelector value={mitreIds} onChange={setMitreIds} />
              </div>

              {/* Type-specific editor */}
              <div className="rounded-xl border p-4">
                <p className="mb-3 text-sm font-semibold capitalize">{selectedType} Rule Configuration</p>
                {selectedType === 'sigma' && (
                  <RuleSigmaEditor
                    value={ruleContent as SigmaRuleContent}
                    onChange={setRuleContent}
                  />
                )}
                {selectedType === 'threshold' && (
                  <RuleThresholdEditor
                    value={ruleContent as ThresholdRuleContent}
                    onChange={setRuleContent}
                  />
                )}
                {selectedType === 'anomaly' && (
                  <RuleAnomalyEditor
                    value={ruleContent as AnomalyRuleContent}
                    onChange={setRuleContent}
                  />
                )}
                {selectedType === 'correlation' && (
                  <RuleCorrelationEditor
                    value={ruleContent as CorrelationRuleContent}
                    onChange={setRuleContent}
                  />
                )}
              </div>

              {/* Preview */}
              <div className="rounded-lg border">
                <button
                  type="button"
                  className="flex w-full items-center justify-between px-4 py-2 text-sm font-medium text-muted-foreground hover:text-foreground"
                  onClick={() => setPreviewOpen((o) => !o)}
                >
                  Preview Rule JSON
                  {previewOpen ? (
                    <ChevronDown className="h-4 w-4" />
                  ) : (
                    <ChevronRight className="h-4 w-4" />
                  )}
                </button>
                {previewOpen && (
                  <pre className="overflow-x-auto rounded-b-lg bg-muted px-4 py-3 text-[11px] leading-relaxed">
                    {JSON.stringify(ruleContent, null, 2)}
                  </pre>
                )}
              </div>
            </form>
          </FormProvider>
        </ScrollArea>

        <DialogFooter className="border-t px-6 py-4">
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="submit" form="rule-form" disabled={isPending}>
            {isPending ? 'Saving…' : isEdit ? 'Save Changes' : 'Create Rule'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
