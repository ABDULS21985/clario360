'use client';

import { useState } from 'react';
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { DSPMDataPolicy, DSPMPolicyCategory, DSPMPolicyEnforcement, CyberSeverity } from '@/types/cyber';

type CreatePolicyPayload = {
  name: string;
  description: string;
  category: DSPMPolicyCategory;
  enforcement: DSPMPolicyEnforcement;
  severity: CyberSeverity;
  rule: Record<string, unknown>;
  scope_classification: string[];
  scope_asset_types: string[];
  enabled: boolean;
  compliance_frameworks: string[];
};

interface PolicyEditorFormProps {
  policy?: DSPMDataPolicy;
  onSubmit: (data: CreatePolicyPayload) => void;
  onCancel: () => void;
}

const CATEGORIES: { value: DSPMPolicyCategory; label: string }[] = [
  { value: 'encryption', label: 'Encryption' },
  { value: 'classification', label: 'Classification' },
  { value: 'retention', label: 'Retention' },
  { value: 'exposure', label: 'Exposure' },
  { value: 'pii_protection', label: 'PII Protection' },
  { value: 'access_review', label: 'Access Review' },
  { value: 'backup', label: 'Backup' },
  { value: 'audit_logging', label: 'Audit Logging' },
];

const ENFORCEMENTS: { value: DSPMPolicyEnforcement; label: string }[] = [
  { value: 'alert', label: 'Alert Only' },
  { value: 'auto_remediate', label: 'Auto Remediate' },
  { value: 'block', label: 'Block' },
];

const SEVERITIES: { value: CyberSeverity; label: string }[] = [
  { value: 'critical', label: 'Critical' },
  { value: 'high', label: 'High' },
  { value: 'medium', label: 'Medium' },
  { value: 'low', label: 'Low' },
  { value: 'info', label: 'Info' },
];

const CLASSIFICATIONS = ['public', 'internal', 'confidential', 'restricted'];
const ASSET_TYPES = ['database', 'cloud_storage', 'file_server', 'api_endpoint', 'data_warehouse', 'object_store'];
const COMPLIANCE_FRAMEWORKS = ['GDPR', 'HIPAA', 'SOC2', 'PCI-DSS', 'Saudi PDPL'];

export function PolicyEditorForm({ policy, onSubmit, onCancel }: PolicyEditorFormProps) {
  const [name, setName] = useState(policy?.name ?? '');
  const [description, setDescription] = useState(policy?.description ?? '');
  const [category, setCategory] = useState<DSPMPolicyCategory>(policy?.category ?? 'encryption');
  const [enforcement, setEnforcement] = useState<DSPMPolicyEnforcement>(policy?.enforcement ?? 'alert');
  const [severity, setSeverity] = useState<CyberSeverity>(policy?.severity ?? 'medium');
  const [enabled, setEnabled] = useState(policy?.enabled ?? true);
  const [rule, setRule] = useState<Record<string, unknown>>(policy?.rule ?? {});
  const [scopeClassification, setScopeClassification] = useState<string[]>(policy?.scope_classification ?? []);
  const [scopeAssetTypes, setScopeAssetTypes] = useState<string[]>(policy?.scope_asset_types ?? []);
  const [complianceFrameworks, setComplianceFrameworks] = useState<string[]>(policy?.compliance_frameworks ?? []);

  const toggleArray = (arr: string[], value: string): string[] =>
    arr.includes(value) ? arr.filter((v) => v !== value) : [...arr, value];

  const updateRule = (key: string, value: unknown) => {
    setRule((prev) => ({ ...prev, [key]: value }));
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit({
      name,
      description,
      category,
      enforcement,
      severity,
      rule,
      scope_classification: scopeClassification,
      scope_asset_types: scopeAssetTypes,
      enabled,
      compliance_frameworks: complianceFrameworks,
    });
  };

  const renderRuleFields = () => {
    switch (category) {
      case 'encryption':
        return (
          <div className="space-y-3">
            <div className="flex items-center gap-2">
              <Checkbox
                id="require_at_rest"
                checked={!!rule.require_at_rest}
                onCheckedChange={(v) => updateRule('require_at_rest', !!v)}
              />
              <Label htmlFor="require_at_rest" className="cursor-pointer">Require encryption at rest</Label>
            </div>
            <div className="flex items-center gap-2">
              <Checkbox
                id="require_in_transit"
                checked={!!rule.require_in_transit}
                onCheckedChange={(v) => updateRule('require_in_transit', !!v)}
              />
              <Label htmlFor="require_in_transit" className="cursor-pointer">Require encryption in transit</Label>
            </div>
          </div>
        );
      case 'classification':
        return (
          <div className="space-y-3">
            <div>
              <Label>Required Classification Level</Label>
              <Select
                value={(rule.required_level as string) ?? ''}
                onValueChange={(v) => updateRule('required_level', v)}
              >
                <SelectTrigger className="mt-1">
                  <SelectValue placeholder="Select level" />
                </SelectTrigger>
                <SelectContent>
                  {CLASSIFICATIONS.map((c) => (
                    <SelectItem key={c} value={c} className="capitalize">{c}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label>Minimum Classification Level</Label>
              <Select
                value={(rule.min_level as string) ?? ''}
                onValueChange={(v) => updateRule('min_level', v)}
              >
                <SelectTrigger className="mt-1">
                  <SelectValue placeholder="Select minimum level" />
                </SelectTrigger>
                <SelectContent>
                  {CLASSIFICATIONS.map((c) => (
                    <SelectItem key={c} value={c} className="capitalize">{c}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
        );
      case 'retention':
        return (
          <div>
            <Label htmlFor="max_retention_days">Maximum Retention (days)</Label>
            <Input
              id="max_retention_days"
              type="number"
              min={1}
              className="mt-1"
              value={(rule.max_retention_days as number) ?? ''}
              onChange={(e) => updateRule('max_retention_days', e.target.value ? Number(e.target.value) : undefined)}
            />
          </div>
        );
      case 'exposure':
        return (
          <div>
            <Label>Maximum Allowed Exposure</Label>
            <Select
              value={(rule.max_exposure as string) ?? ''}
              onValueChange={(v) => updateRule('max_exposure', v)}
            >
              <SelectTrigger className="mt-1">
                <SelectValue placeholder="Select max exposure" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="private">Private</SelectItem>
                <SelectItem value="internal">Internal</SelectItem>
                <SelectItem value="dmz">DMZ</SelectItem>
                <SelectItem value="internet_facing">Internet Facing</SelectItem>
              </SelectContent>
            </Select>
          </div>
        );
      case 'pii_protection':
        return (
          <div className="space-y-3">
            <div className="flex items-center gap-2">
              <Checkbox
                id="require_encryption_pii"
                checked={!!rule.require_encryption}
                onCheckedChange={(v) => updateRule('require_encryption', !!v)}
              />
              <Label htmlFor="require_encryption_pii" className="cursor-pointer">Require encryption for PII</Label>
            </div>
            <div className="flex items-center gap-2">
              <Checkbox
                id="require_masking"
                checked={!!rule.require_masking}
                onCheckedChange={(v) => updateRule('require_masking', !!v)}
              />
              <Label htmlFor="require_masking" className="cursor-pointer">Require data masking</Label>
            </div>
            <div>
              <Label htmlFor="allowed_pii_types">Allowed PII Types (comma-separated)</Label>
              <Input
                id="allowed_pii_types"
                className="mt-1"
                placeholder="e.g. email, phone, name"
                value={Array.isArray(rule.allowed_pii_types) ? (rule.allowed_pii_types as string[]).join(', ') : ''}
                onChange={(e) =>
                  updateRule(
                    'allowed_pii_types',
                    e.target.value
                      .split(',')
                      .map((s) => s.trim())
                      .filter(Boolean),
                  )
                }
              />
            </div>
          </div>
        );
      case 'access_review':
        return (
          <div>
            <Label htmlFor="max_days_since_review">Max Days Since Last Review</Label>
            <Input
              id="max_days_since_review"
              type="number"
              min={1}
              className="mt-1"
              value={(rule.max_days_since_review as number) ?? ''}
              onChange={(e) => updateRule('max_days_since_review', e.target.value ? Number(e.target.value) : undefined)}
            />
          </div>
        );
      case 'backup':
        return (
          <div className="flex items-center gap-2">
            <Checkbox
              id="require_backup"
              checked={!!rule.require_backup}
              onCheckedChange={(v) => updateRule('require_backup', !!v)}
            />
            <Label htmlFor="require_backup" className="cursor-pointer">Require backup</Label>
          </div>
        );
      case 'audit_logging':
        return (
          <div className="flex items-center gap-2">
            <Checkbox
              id="require_audit"
              checked={!!rule.require_audit}
              onCheckedChange={(v) => updateRule('require_audit', !!v)}
            />
            <Label htmlFor="require_audit" className="cursor-pointer">Require audit logging</Label>
          </div>
        );
      default:
        return null;
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {/* Basic Info */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{policy ? 'Edit Policy' : 'Create Policy'}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <Label htmlFor="policy-name">Policy Name</Label>
            <Input
              id="policy-name"
              className="mt-1"
              placeholder="Enter policy name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
            />
          </div>
          <div>
            <Label htmlFor="policy-description">Description</Label>
            <Textarea
              id="policy-description"
              className="mt-1"
              placeholder="Describe what this policy enforces..."
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
            />
          </div>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <div>
              <Label>Category</Label>
              <Select value={category} onValueChange={(v) => { setCategory(v as DSPMPolicyCategory); setRule({}); }}>
                <SelectTrigger className="mt-1">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {CATEGORIES.map((c) => (
                    <SelectItem key={c.value} value={c.value}>{c.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label>Enforcement</Label>
              <Select value={enforcement} onValueChange={(v) => setEnforcement(v as DSPMPolicyEnforcement)}>
                <SelectTrigger className="mt-1">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {ENFORCEMENTS.map((e) => (
                    <SelectItem key={e.value} value={e.value}>{e.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label>Severity</Label>
              <Select value={severity} onValueChange={(v) => setSeverity(v as CyberSeverity)}>
                <SelectTrigger className="mt-1">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {SEVERITIES.map((s) => (
                    <SelectItem key={s.value} value={s.value}>{s.label}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Checkbox
              id="policy-enabled"
              checked={enabled}
              onCheckedChange={(v) => setEnabled(!!v)}
            />
            <Label htmlFor="policy-enabled" className="cursor-pointer">Policy enabled</Label>
          </div>
        </CardContent>
      </Card>

      {/* Rule Builder */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Rule Configuration</CardTitle>
        </CardHeader>
        <CardContent>{renderRuleFields()}</CardContent>
      </Card>

      {/* Scope */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Scope</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <p className="mb-2 text-sm font-medium">Classification Filter</p>
            <div className="flex flex-wrap gap-3">
              {CLASSIFICATIONS.map((cls) => (
                <div key={cls} className="flex items-center gap-2">
                  <Checkbox
                    id={`scope-cls-${cls}`}
                    checked={scopeClassification.includes(cls)}
                    onCheckedChange={() => setScopeClassification(toggleArray(scopeClassification, cls))}
                  />
                  <Label htmlFor={`scope-cls-${cls}`} className="cursor-pointer capitalize">{cls}</Label>
                </div>
              ))}
            </div>
          </div>
          <div>
            <p className="mb-2 text-sm font-medium">Asset Type Filter</p>
            <div className="flex flex-wrap gap-3">
              {ASSET_TYPES.map((at) => (
                <div key={at} className="flex items-center gap-2">
                  <Checkbox
                    id={`scope-at-${at}`}
                    checked={scopeAssetTypes.includes(at)}
                    onCheckedChange={() => setScopeAssetTypes(toggleArray(scopeAssetTypes, at))}
                  />
                  <Label htmlFor={`scope-at-${at}`} className="cursor-pointer capitalize">
                    {at.replace(/_/g, ' ')}
                  </Label>
                </div>
              ))}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Compliance Frameworks */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Compliance Frameworks</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-wrap gap-3">
            {COMPLIANCE_FRAMEWORKS.map((fw) => (
              <div key={fw} className="flex items-center gap-2">
                <Checkbox
                  id={`fw-${fw}`}
                  checked={complianceFrameworks.includes(fw)}
                  onCheckedChange={() => setComplianceFrameworks(toggleArray(complianceFrameworks, fw))}
                />
                <Label htmlFor={`fw-${fw}`} className="cursor-pointer">{fw}</Label>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Actions */}
      <div className="flex items-center justify-end gap-3">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit">
          {policy ? 'Update Policy' : 'Create Policy'}
        </Button>
      </div>
    </form>
  );
}
