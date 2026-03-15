'use client';

import { useEffect, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { ChevronRight } from 'lucide-react';
import { toast } from 'sonner';

import { useApiMutation } from '@/hooks/use-api-mutation';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import {
  defaultAnomalyContent,
  defaultCorrelationContent,
  defaultSigmaContent,
  defaultThresholdContent,
  DETECTION_RULE_TYPE_OPTIONS,
  getRuleTypeLabel,
  normalizeRule,
  parseSigmaYamlText,
  RULE_SEVERITY_OPTIONS,
  serializeRuleContent,
  stringifySigmaContent,
} from '@/lib/cyber-rules';
import type {
  AnomalyRuleContent,
  CorrelationRuleContent,
  CyberSeverity,
  DetectionRule,
  DetectionRuleType,
  MITRETacticItem,
  MITRETechniqueItem,
  SigmaRuleContent,
  ThresholdRuleContent,
} from '@/types/cyber';
import { slugToTitle } from '@/lib/utils';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Separator } from '@/components/ui/separator';
import { Switch } from '@/components/ui/switch';
import { Textarea } from '@/components/ui/textarea';
import { Checkbox } from '@/components/ui/checkbox';

import { RuleAnomalyEditor } from './rule-anomaly-editor';
import { RuleCorrelationEditor } from './rule-correlation-editor';
import { RuleSigmaMonaco } from './rule-sigma-monaco';
import { RuleThresholdEditor } from './rule-threshold-editor';

type WizardStep = 0 | 1 | 2 | 3;

interface RuleWizardProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: () => void;
  rule?: DetectionRule | null;
  initialTechniqueId?: string | null;
}

const STEPS = ['Basics', 'Detection Logic', 'MITRE Mapping', 'Review'] as const;

export function RuleWizard({
  open,
  onOpenChange,
  onSuccess,
  rule,
  initialTechniqueId,
}: RuleWizardProps) {
  const sourceRule = rule ? normalizeRule(rule) : null;
  const editingRule = sourceRule?.id ? sourceRule : null;
  const [step, setStep] = useState<WizardStep>(0);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [ruleType, setRuleType] = useState<DetectionRuleType>('sigma');
  const [severity, setSeverity] = useState<CyberSeverity>('medium');
  const [enabled, setEnabled] = useState(true);
  const [baseConfidence, setBaseConfidence] = useState(0.7);
  const [tagsInput, setTagsInput] = useState('');
  const [sigmaYaml, setSigmaYaml] = useState('');
  const [thresholdContent, setThresholdContent] = useState<ThresholdRuleContent>(defaultThresholdContent());
  const [correlationContent, setCorrelationContent] = useState<CorrelationRuleContent>(defaultCorrelationContent());
  const [anomalyContent, setAnomalyContent] = useState<AnomalyRuleContent>(defaultAnomalyContent());
  const [selectedTactics, setSelectedTactics] = useState<string[]>([]);
  const [selectedTechniques, setSelectedTechniques] = useState<string[]>([]);
  const [validationError, setValidationError] = useState<string | null>(null);

  const { data: tacticsEnvelope } = useQuery({
    queryKey: ['cyber-rule-mitre-tactics'],
    queryFn: () => apiGet<{ data: MITRETacticItem[] }>(API_ENDPOINTS.CYBER_MITRE_TACTICS),
    staleTime: 300_000,
  });

  const { data: techniquesEnvelope } = useQuery({
    queryKey: ['cyber-rule-mitre-techniques'],
    queryFn: () => apiGet<{ data: MITRETechniqueItem[] }>(API_ENDPOINTS.CYBER_MITRE_TECHNIQUES),
    staleTime: 300_000,
  });

  const tactics = tacticsEnvelope?.data ?? [];
  const techniques = techniquesEnvelope?.data ?? [];

  const groupedTechniques = useMemo(() => {
    return tactics.map((tactic) => ({
      tactic,
      techniques: techniques.filter((technique) => technique.tactic_ids.includes(tactic.id)),
    }));
  }, [tactics, techniques]);

  useEffect(() => {
    if (!open) {
      return;
    }

    setStep(0);
    setValidationError(null);
    setName(sourceRule?.name ?? '');
    setDescription(sourceRule?.description ?? '');
    setRuleType(sourceRule?.rule_type ?? 'sigma');
    setSeverity(sourceRule?.severity ?? 'medium');
    setEnabled(sourceRule?.enabled ?? true);
    setBaseConfidence(sourceRule?.base_confidence ?? 0.7);
    setTagsInput((sourceRule?.tags ?? []).join(', '));
    setThresholdContent((sourceRule?.rule_type === 'threshold' ? sourceRule.rule_content : defaultThresholdContent()) as ThresholdRuleContent);
    setCorrelationContent((sourceRule?.rule_type === 'correlation' ? sourceRule.rule_content : defaultCorrelationContent()) as CorrelationRuleContent);
    setAnomalyContent((sourceRule?.rule_type === 'anomaly' ? sourceRule.rule_content : defaultAnomalyContent()) as AnomalyRuleContent);

    const sigmaContent =
      sourceRule?.rule_type === 'sigma'
        ? serializeRuleContent('sigma', sourceRule.rule_content as SigmaRuleContent)
        : serializeRuleContent('sigma', defaultSigmaContent());
    setSigmaYaml(stringifySigmaContent(sigmaContent));

    const initialTechniques = sourceRule?.mitre_technique_ids ?? (initialTechniqueId ? [initialTechniqueId] : []);
    setSelectedTechniques(initialTechniques);

    const techniqueTactics = new Set<string>(sourceRule?.mitre_tactic_ids ?? []);
    initialTechniques.forEach((techniqueId) => {
      techniques.find((technique) => technique.id === techniqueId)?.tactic_ids.forEach((tacticId) => {
        techniqueTactics.add(tacticId);
      });
    });
    setSelectedTactics(Array.from(techniqueTactics));
  }, [initialTechniqueId, open, sourceRule, techniques]);

  useEffect(() => {
    setSelectedTechniques((current) =>
      current.filter((techniqueId) => {
        const technique = techniques.find((item) => item.id === techniqueId);
        return !technique || technique.tactic_ids.some((tacticId) => selectedTactics.includes(tacticId));
      }),
    );
  }, [selectedTactics, techniques]);

  const mutation = useApiMutation<{ data: DetectionRule }, Record<string, unknown>>(
    editingRule ? 'put' : 'post',
    editingRule ? API_ENDPOINTS.CYBER_RULE_DETAIL(editingRule.id) : API_ENDPOINTS.CYBER_RULES,
    {
      successMessage: editingRule ? 'Detection rule updated' : 'Detection rule created',
      invalidateKeys: ['cyber-rules', 'cyber-rules-stats', 'cyber-mitre-coverage'],
      onSuccess: () => {
        onOpenChange(false);
        onSuccess?.();
      },
    },
  );

  const currentLogic = useMemo(() => {
    if (ruleType === 'threshold') {
      return thresholdContent;
    }
    if (ruleType === 'correlation') {
      return correlationContent;
    }
    if (ruleType === 'anomaly') {
      return anomalyContent;
    }
    return null;
  }, [anomalyContent, correlationContent, ruleType, thresholdContent]);

  function validateCurrentStep(targetStep: WizardStep): boolean {
    if (targetStep === 0) {
      if (name.trim().length < 3) {
        setValidationError('Rule name must be at least 3 characters.');
        return false;
      }
      setValidationError(null);
      return true;
    }

    if (targetStep === 1 && ruleType === 'sigma') {
      try {
        parseSigmaYamlText(sigmaYaml);
        setValidationError(null);
        return true;
      } catch (error) {
        setValidationError(error instanceof Error ? error.message : 'Invalid Sigma YAML');
        return false;
      }
    }

    if (targetStep === 1 && ruleType === 'threshold' && !thresholdContent.group_by) {
      setValidationError('Threshold rules require a group-by field.');
      return false;
    }

    if (targetStep === 1 && ruleType === 'correlation' && !correlationContent.group_by) {
      setValidationError('Correlation rules require a correlation field.');
      return false;
    }

    setValidationError(null);
    return true;
  }

  function goNext() {
    if (!validateCurrentStep(step)) {
      return;
    }
    setStep((current) => (current >= 3 ? 3 : ((current + 1) as WizardStep)));
  }

  function toggleTactic(tacticId: string) {
    setSelectedTactics((current) =>
      current.includes(tacticId)
        ? current.filter((item) => item !== tacticId)
        : [...current, tacticId],
    );
  }

  function toggleTechnique(techniqueId: string, tacticIds: string[]) {
    setSelectedTechniques((current) =>
      current.includes(techniqueId)
        ? current.filter((item) => item !== techniqueId)
        : [...current, techniqueId],
    );
    setSelectedTactics((current) => Array.from(new Set([...current, ...tacticIds])));
  }

  function handleSave() {
    if (!validateCurrentStep(step)) {
      return;
    }

    try {
      const ruleContent =
        ruleType === 'sigma'
          ? parseSigmaYamlText(sigmaYaml)
          : serializeRuleContent(ruleType, currentLogic as ThresholdRuleContent | CorrelationRuleContent | AnomalyRuleContent);

      mutation.mutate({
        name: name.trim(),
        description: description.trim(),
        rule_type: ruleType,
        severity,
        enabled,
        base_confidence: baseConfidence,
        rule_content: ruleContent,
        mitre_tactic_ids: selectedTactics,
        mitre_technique_ids: selectedTechniques,
        tags: tagsInput
          .split(',')
          .map((tag) => tag.trim())
          .filter(Boolean),
      });
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unable to build rule payload';
      setValidationError(message);
      toast.error(message);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[92vh] overflow-hidden p-0 sm:max-w-5xl">
        <DialogHeader className="border-b px-6 py-5">
          <DialogTitle>{editingRule ? 'Edit Detection Rule' : 'Create Detection Rule'}</DialogTitle>
        </DialogHeader>

        <div className="flex flex-col gap-6 overflow-y-auto px-6 py-5">
          <div className="grid grid-cols-1 gap-3 md:grid-cols-4">
            {STEPS.map((label, index) => (
              <button
                key={label}
                type="button"
                onClick={() => {
                  if (index <= step || validateCurrentStep(step)) {
                    setStep(index as WizardStep);
                  }
                }}
                className={`rounded-[22px] border px-4 py-3 text-left transition ${
                  step === index ? 'border-emerald-300 bg-emerald-50 text-emerald-950' : 'border-border bg-white text-muted-foreground'
                }`}
              >
                <p className="text-[11px] font-semibold uppercase tracking-[0.18em]">Step {index + 1}</p>
                <p className="mt-2 text-sm font-medium">{label}</p>
              </button>
            ))}
          </div>

          {step === 0 && (
            <div className="grid grid-cols-1 gap-6 lg:grid-cols-[1.4fr_1fr]">
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="rule-name">Rule name</Label>
                  <Input id="rule-name" value={name} onChange={(event) => setName(event.target.value)} placeholder="Suspicious PowerShell execution" />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="rule-description">Description</Label>
                  <Textarea
                    id="rule-description"
                    rows={5}
                    value={description}
                    onChange={(event) => setDescription(event.target.value)}
                    placeholder="Explain what the rule detects and why it matters."
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="rule-tags">Tags</Label>
                  <Input
                    id="rule-tags"
                    value={tagsInput}
                    onChange={(event) => setTagsInput(event.target.value)}
                    placeholder="powershell, credential-access, endpoint"
                  />
                </div>
              </div>

              <div className="space-y-4 rounded-[26px] border border-[color:var(--card-border)] bg-[var(--card-bg)] p-5 shadow-[var(--card-shadow)]">
                <div className="space-y-2">
                  <Label>Rule type</Label>
                  <Select value={ruleType} onValueChange={(value) => setRuleType(value as DetectionRuleType)}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {DETECTION_RULE_TYPE_OPTIONS.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          {option.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-2">
                  <Label>Severity</Label>
                  <Select value={severity} onValueChange={(value) => setSeverity(value as CyberSeverity)}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {RULE_SEVERITY_OPTIONS.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          {option.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="rule-confidence">Base confidence</Label>
                  <Input
                    id="rule-confidence"
                    type="number"
                    min={0}
                    max={1}
                    step={0.05}
                    value={baseConfidence}
                    onChange={(event) => setBaseConfidence(Number(event.target.value) || 0)}
                  />
                </div>

                <div className="flex items-center justify-between rounded-2xl border px-4 py-3">
                  <div>
                    <p className="text-sm font-medium">Rule enabled</p>
                    <p className="text-xs text-muted-foreground">Inactive rules stay available without generating detections.</p>
                  </div>
                  <Switch checked={enabled} onCheckedChange={setEnabled} />
                </div>
              </div>
            </div>
          )}

          {step === 1 && (
            <div className="space-y-4">
              {ruleType === 'sigma' ? (
                <div className="space-y-3">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <p className="text-sm font-medium">Sigma YAML</p>
                      <p className="text-sm text-muted-foreground">Write a Sigma-style detection block. The wizard validates the YAML before saving.</p>
                    </div>
                    <Badge variant="outline">Monaco</Badge>
                  </div>
                  <RuleSigmaMonaco value={sigmaYaml} onChange={setSigmaYaml} />
                </div>
              ) : null}

              {ruleType === 'threshold' ? (
                <RuleThresholdEditor value={thresholdContent} onChange={setThresholdContent} />
              ) : null}

              {ruleType === 'correlation' ? (
                <div className="space-y-4">
                  <RuleCorrelationEditor value={correlationContent} onChange={setCorrelationContent} />
                  <div className="space-y-2">
                    <Label htmlFor="min-failed-count">Minimum first-event matches</Label>
                    <Input
                      id="min-failed-count"
                      type="number"
                      min={0}
                      value={correlationContent.min_failed_count ?? 0}
                      onChange={(event) =>
                        setCorrelationContent((current) => ({
                          ...current,
                          min_failed_count: Number(event.target.value) || 0,
                        }))
                      }
                    />
                  </div>
                </div>
              ) : null}

              {ruleType === 'anomaly' ? (
                <RuleAnomalyEditor value={anomalyContent} onChange={setAnomalyContent} />
              ) : null}
            </div>
          )}

          {step === 2 && (
            <div className="grid grid-cols-1 gap-6 lg:grid-cols-[280px_1fr]">
              <div className="rounded-[26px] border border-[color:var(--card-border)] bg-[var(--card-bg)] p-4 shadow-[var(--card-shadow)]">
                <p className="text-sm font-medium">Tactics</p>
                <p className="mt-1 text-sm text-muted-foreground">Select the ATT&CK tactics this rule is designed to cover.</p>
                <div className="mt-4 space-y-2">
                  {tactics.map((tactic) => (
                    <label key={tactic.id} className="flex items-start gap-3 rounded-2xl border px-3 py-3">
                      <Checkbox
                        checked={selectedTactics.includes(tactic.id)}
                        onCheckedChange={() => toggleTactic(tactic.id)}
                      />
                      <div>
                        <p className="text-sm font-medium">{tactic.name}</p>
                        <p className="text-xs text-muted-foreground">{tactic.id}</p>
                      </div>
                    </label>
                  ))}
                </div>
              </div>

              <div className="rounded-[26px] border border-[color:var(--card-border)] bg-[var(--card-bg)] p-4 shadow-[var(--card-shadow)]">
                <p className="text-sm font-medium">Techniques</p>
                <p className="mt-1 text-sm text-muted-foreground">Choose techniques from the selected tactics.</p>
                <div className="mt-4 grid grid-cols-1 gap-4 xl:grid-cols-2">
                  {groupedTechniques
                    .filter((group) => selectedTactics.includes(group.tactic.id))
                    .map((group) => (
                      <div key={group.tactic.id} className="rounded-2xl border p-4">
                        <div className="mb-3">
                          <p className="text-sm font-medium">{group.tactic.name}</p>
                          <p className="text-xs text-muted-foreground">{group.tactic.id}</p>
                        </div>
                        <div className="space-y-2">
                          {group.techniques.map((technique) => (
                            <label key={technique.id} className="flex items-start gap-3 rounded-xl border px-3 py-3">
                              <Checkbox
                                checked={selectedTechniques.includes(technique.id)}
                                onCheckedChange={() => toggleTechnique(technique.id, technique.tactic_ids)}
                              />
                              <div>
                                <p className="text-sm font-medium">
                                  <span className="mr-2 font-mono text-xs text-muted-foreground">{technique.id}</span>
                                  {technique.name}
                                </p>
                                <p className="text-xs text-muted-foreground">{technique.description}</p>
                              </div>
                            </label>
                          ))}
                        </div>
                      </div>
                    ))}
                </div>
              </div>
            </div>
          )}

          {step === 3 && (
            <div className="grid grid-cols-1 gap-6 lg:grid-cols-[1.2fr_1fr]">
              <div className="rounded-[26px] border border-[color:var(--card-border)] bg-[var(--card-bg)] p-5 shadow-[var(--card-shadow)]">
                <p className="text-sm font-medium">Configuration summary</p>
                <div className="mt-4 space-y-4">
                  <div>
                    <p className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">Basics</p>
                    <p className="mt-2 text-lg font-semibold">{name || 'Untitled rule'}</p>
                    <p className="text-sm text-muted-foreground">{description || 'No description provided.'}</p>
                  </div>

                  <Separator />

                  <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
                    <div>
                      <p className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">Type</p>
                      <p className="mt-2 text-sm font-medium">{getRuleTypeLabel(ruleType)}</p>
                    </div>
                    <div>
                      <p className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">Severity</p>
                      <p className="mt-2 text-sm font-medium">{slugToTitle(severity)}</p>
                    </div>
                    <div>
                      <p className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">Confidence</p>
                      <p className="mt-2 text-sm font-medium">{Math.round(baseConfidence * 100)}%</p>
                    </div>
                  </div>

                  <Separator />

                  <div>
                    <p className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">MITRE Mapping</p>
                    <div className="mt-3 flex flex-wrap gap-2">
                      {selectedTechniques.length === 0 ? (
                        <span className="text-sm text-muted-foreground">No techniques selected.</span>
                      ) : (
                        selectedTechniques.map((techniqueId) => (
                          <Badge key={techniqueId} variant="outline" className="font-mono">
                            {techniqueId}
                          </Badge>
                        ))
                      )}
                    </div>
                  </div>
                </div>
              </div>

              <div className="rounded-[26px] border border-[color:var(--card-border)] bg-[var(--card-bg)] p-5 shadow-[var(--card-shadow)]">
                <p className="text-sm font-medium">Detection logic preview</p>
                <pre className="mt-4 max-h-[420px] overflow-auto rounded-2xl bg-slate-950/95 p-4 text-xs text-slate-100">
                  {ruleType === 'sigma'
                    ? sigmaYaml
                    : JSON.stringify(
                        serializeRuleContent(ruleType, currentLogic as ThresholdRuleContent | CorrelationRuleContent | AnomalyRuleContent),
                        null,
                        2,
                      )}
                </pre>
              </div>
            </div>
          )}

          {validationError ? (
            <div className="rounded-2xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
              {validationError}
            </div>
          ) : null}
        </div>

        <DialogFooter className="border-t px-6 py-4">
          <div className="flex w-full items-center justify-between">
            <Button
              variant="outline"
              onClick={() =>
                step === 0
                  ? onOpenChange(false)
                  : setStep((current) => (current <= 0 ? 0 : ((current - 1) as WizardStep)))
              }
            >
              {step === 0 ? 'Cancel' : 'Back'}
            </Button>

            {step < 3 ? (
              <Button onClick={goNext}>
                Next
                <ChevronRight className="ml-2 h-4 w-4" />
              </Button>
            ) : (
              <Button onClick={handleSave} disabled={mutation.isPending}>
                {mutation.isPending ? 'Saving…' : editingRule ? 'Update Rule' : 'Create Rule'}
              </Button>
            )}
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
