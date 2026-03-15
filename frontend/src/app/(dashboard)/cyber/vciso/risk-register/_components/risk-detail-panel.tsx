'use client';

import { useState } from 'react';
import { toast } from 'sonner';
import {
  Shield,
  Calendar,
  User,
  Building2,
  Tag,
  FileText,
  CheckCircle,
  Clock,
  Edit,
  Save,
  X,
} from 'lucide-react';
import { DetailPanel } from '@/components/shared/detail-panel';
import { StatusBadge } from '@/components/shared/status-badge';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Separator } from '@/components/ui/separator';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import { formatDate } from '@/lib/format';
import { cn } from '@/lib/utils';
import { riskStatusConfig, riskTreatmentConfig } from '@/lib/status-configs';
import type {
  VCISORiskEntry,
  RiskLikelihood,
  RiskImpact,
  RiskStatus,
  RiskTreatment,
} from '@/types/cyber';

interface RiskDetailPanelProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  risk: VCISORiskEntry;
  onUpdated: () => void;
}

const LIKELIHOOD_OPTIONS: { label: string; value: RiskLikelihood }[] = [
  { label: 'Rare', value: 'rare' },
  { label: 'Unlikely', value: 'unlikely' },
  { label: 'Possible', value: 'possible' },
  { label: 'Likely', value: 'likely' },
  { label: 'Almost Certain', value: 'almost_certain' },
];

const IMPACT_OPTIONS: { label: string; value: RiskImpact }[] = [
  { label: 'Negligible', value: 'negligible' },
  { label: 'Minor', value: 'minor' },
  { label: 'Moderate', value: 'moderate' },
  { label: 'Major', value: 'major' },
  { label: 'Catastrophic', value: 'catastrophic' },
];

const STATUS_OPTIONS: { label: string; value: RiskStatus }[] = [
  { label: 'Identified', value: 'identified' },
  { label: 'Assessed', value: 'assessed' },
  { label: 'Mitigating', value: 'mitigating' },
  { label: 'Accepted', value: 'accepted' },
  { label: 'Closed', value: 'closed' },
];

const TREATMENT_OPTIONS: { label: string; value: RiskTreatment }[] = [
  { label: 'Mitigate', value: 'mitigate' },
  { label: 'Transfer', value: 'transfer' },
  { label: 'Accept', value: 'accept' },
  { label: 'Avoid', value: 'avoid' },
];

function scoreColor(score: number): string {
  if (score <= 30) return 'text-green-600';
  if (score <= 60) return 'text-amber-600';
  return 'text-red-600';
}

function scoreBgColor(score: number): string {
  if (score <= 30) return 'bg-green-100 text-green-700';
  if (score <= 60) return 'bg-amber-100 text-amber-700';
  return 'bg-red-100 text-red-700';
}

function titleCase(str: string): string {
  return str
    .replace(/[-_]/g, ' ')
    .split(' ')
    .map((w) => (w.length > 0 ? w.charAt(0).toUpperCase() + w.slice(1).toLowerCase() : w))
    .join(' ');
}

export function RiskDetailPanel({
  open,
  onOpenChange,
  risk,
  onUpdated,
}: RiskDetailPanelProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [editForm, setEditForm] = useState({
    title: risk.title,
    description: risk.description,
    category: risk.category,
    likelihood: risk.likelihood,
    impact: risk.impact,
    status: risk.status,
    treatment: risk.treatment,
    department: risk.department,
    treatment_plan: risk.treatment_plan,
    business_services: risk.business_services.join(', '),
    controls: risk.controls.join(', '),
    review_date: risk.review_date ? risk.review_date.split('T')[0] : '',
  });

  const updateMutation = useApiMutation<VCISORiskEntry, Record<string, unknown>>(
    'put',
    `${API_ENDPOINTS.CYBER_VCISO_RISKS}/${risk.id}`,
    {
      successMessage: 'Risk updated successfully',
      invalidateKeys: ['vciso-risks', API_ENDPOINTS.CYBER_VCISO_RISKS_STATS],
      onSuccess: () => {
        setIsEditing(false);
        onUpdated();
      },
    },
  );

  const handleSave = () => {
    if (!editForm.title.trim()) {
      toast.error('Title is required');
      return;
    }

    updateMutation.mutate({
      title: editForm.title.trim(),
      description: editForm.description.trim(),
      category: editForm.category.trim(),
      likelihood: editForm.likelihood,
      impact: editForm.impact,
      status: editForm.status,
      treatment: editForm.treatment,
      department: editForm.department.trim(),
      treatment_plan: editForm.treatment_plan.trim(),
      business_services: editForm.business_services
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean),
      controls: editForm.controls
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean),
      review_date: editForm.review_date || undefined,
    });
  };

  const handleCancel = () => {
    setIsEditing(false);
    setEditForm({
      title: risk.title,
      description: risk.description,
      category: risk.category,
      likelihood: risk.likelihood,
      impact: risk.impact,
      status: risk.status,
      treatment: risk.treatment,
      department: risk.department,
      treatment_plan: risk.treatment_plan,
      business_services: risk.business_services.join(', '),
      controls: risk.controls.join(', '),
      review_date: risk.review_date ? risk.review_date.split('T')[0] : '',
    });
  };

  const handleOpenChange = (o: boolean) => {
    if (!o) {
      setIsEditing(false);
    }
    onOpenChange(o);
  };

  return (
    <DetailPanel
      open={open}
      onOpenChange={handleOpenChange}
      title={isEditing ? 'Edit Risk' : risk.title}
      description={isEditing ? 'Update risk details below' : risk.category}
      width="xl"
    >
      <div className="space-y-6">
        {/* Action bar */}
        <div className="flex items-center justify-end gap-2">
          {isEditing ? (
            <>
              <Button
                variant="outline"
                size="sm"
                onClick={handleCancel}
                disabled={updateMutation.isPending}
              >
                <X className="mr-1.5 h-4 w-4" />
                Cancel
              </Button>
              <Button
                size="sm"
                onClick={handleSave}
                disabled={updateMutation.isPending}
              >
                <Save className="mr-1.5 h-4 w-4" />
                {updateMutation.isPending ? 'Saving...' : 'Save'}
              </Button>
            </>
          ) : (
            <Button variant="outline" size="sm" onClick={() => setIsEditing(true)}>
              <Edit className="mr-1.5 h-4 w-4" />
              Edit
            </Button>
          )}
        </div>

        {isEditing ? (
          /* ── Edit form ────────────────────────────────────────── */
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="edit-title">Title</Label>
              <Input
                id="edit-title"
                value={editForm.title}
                onChange={(e) => setEditForm((f) => ({ ...f, title: e.target.value }))}
                placeholder="Risk title"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="edit-description">Description</Label>
              <Textarea
                id="edit-description"
                value={editForm.description}
                onChange={(e) => setEditForm((f) => ({ ...f, description: e.target.value }))}
                placeholder="Describe the risk"
                rows={3}
              />
            </div>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="edit-category">Category</Label>
                <Input
                  id="edit-category"
                  value={editForm.category}
                  onChange={(e) => setEditForm((f) => ({ ...f, category: e.target.value }))}
                  placeholder="e.g. Operational"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="edit-department">Department</Label>
                <Input
                  id="edit-department"
                  value={editForm.department}
                  onChange={(e) => setEditForm((f) => ({ ...f, department: e.target.value }))}
                  placeholder="e.g. Engineering"
                />
              </div>
            </div>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label>Likelihood</Label>
                <Select
                  value={editForm.likelihood}
                  onValueChange={(v) => setEditForm((f) => ({ ...f, likelihood: v as RiskLikelihood }))}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {LIKELIHOOD_OPTIONS.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value}>
                        {opt.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>Impact</Label>
                <Select
                  value={editForm.impact}
                  onValueChange={(v) => setEditForm((f) => ({ ...f, impact: v as RiskImpact }))}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {IMPACT_OPTIONS.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value}>
                        {opt.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label>Status</Label>
                <Select
                  value={editForm.status}
                  onValueChange={(v) => setEditForm((f) => ({ ...f, status: v as RiskStatus }))}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {STATUS_OPTIONS.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value}>
                        {opt.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>Treatment</Label>
                <Select
                  value={editForm.treatment}
                  onValueChange={(v) => setEditForm((f) => ({ ...f, treatment: v as RiskTreatment }))}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {TREATMENT_OPTIONS.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value}>
                        {opt.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="edit-review-date">Review Date</Label>
              <Input
                id="edit-review-date"
                type="date"
                value={editForm.review_date}
                onChange={(e) => setEditForm((f) => ({ ...f, review_date: e.target.value }))}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="edit-treatment-plan">Treatment Plan</Label>
              <Textarea
                id="edit-treatment-plan"
                value={editForm.treatment_plan}
                onChange={(e) => setEditForm((f) => ({ ...f, treatment_plan: e.target.value }))}
                placeholder="Describe the treatment plan"
                rows={3}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="edit-controls">Controls (comma-separated)</Label>
              <Textarea
                id="edit-controls"
                value={editForm.controls}
                onChange={(e) => setEditForm((f) => ({ ...f, controls: e.target.value }))}
                placeholder="AC-1, AC-2, AC-3"
                rows={2}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="edit-services">Business Services (comma-separated)</Label>
              <Textarea
                id="edit-services"
                value={editForm.business_services}
                onChange={(e) => setEditForm((f) => ({ ...f, business_services: e.target.value }))}
                placeholder="Payment Processing, Customer Portal"
                rows={2}
              />
            </div>
          </div>
        ) : (
          /* ── Read-only view ───────────────────────────────────── */
          <>
            {/* Overview section */}
            <div className="space-y-3">
              <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
                Overview
              </h3>
              <p className="text-sm text-foreground leading-relaxed">
                {risk.description || 'No description provided.'}
              </p>
              <div className="flex flex-wrap gap-2">
                <StatusBadge status={risk.status} config={riskStatusConfig} />
                <StatusBadge status={risk.treatment} config={riskTreatmentConfig} />
                <Badge variant="outline">{risk.category}</Badge>
              </div>
            </div>

            <Separator />

            {/* Scores section */}
            <div className="space-y-3">
              <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
                Risk Scores
              </h3>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div className="rounded-xl border p-4 text-center">
                  <p className="text-xs text-muted-foreground mb-1">Inherent Score</p>
                  <p className={cn('text-2xl font-bold', scoreColor(risk.inherent_score))}>
                    {risk.inherent_score}
                  </p>
                </div>
                <div className="rounded-xl border p-4 text-center">
                  <p className="text-xs text-muted-foreground mb-1">Residual Score</p>
                  <p className={cn('text-2xl font-bold', scoreColor(risk.residual_score))}>
                    {risk.residual_score}
                  </p>
                </div>
              </div>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div className="flex items-center gap-2 text-sm">
                  <span className="text-muted-foreground">Likelihood:</span>
                  <span className="font-medium">{titleCase(risk.likelihood)}</span>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <span className="text-muted-foreground">Impact:</span>
                  <span className="font-medium">{titleCase(risk.impact)}</span>
                </div>
              </div>
            </div>

            <Separator />

            {/* Assignment & Timeline */}
            <div className="space-y-3">
              <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
                Assignment & Timeline
              </h3>
              <div className="space-y-2">
                <div className="flex items-center gap-2 text-sm">
                  <User className="h-4 w-4 text-muted-foreground" />
                  <span className="text-muted-foreground">Owner:</span>
                  <span className="font-medium">{risk.owner_name || 'Unassigned'}</span>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <Building2 className="h-4 w-4 text-muted-foreground" />
                  <span className="text-muted-foreground">Department:</span>
                  <span className="font-medium">{risk.department || 'N/A'}</span>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <Calendar className="h-4 w-4 text-muted-foreground" />
                  <span className="text-muted-foreground">Review Date:</span>
                  <span className="font-medium">
                    {risk.review_date ? formatDate(risk.review_date) : 'Not set'}
                  </span>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <Clock className="h-4 w-4 text-muted-foreground" />
                  <span className="text-muted-foreground">Created:</span>
                  <span className="font-medium">{formatDate(risk.created_at)}</span>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <Clock className="h-4 w-4 text-muted-foreground" />
                  <span className="text-muted-foreground">Updated:</span>
                  <span className="font-medium">{formatDate(risk.updated_at)}</span>
                </div>
              </div>
            </div>

            <Separator />

            {/* Treatment Plan */}
            <div className="space-y-3">
              <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
                Treatment Plan
              </h3>
              <p className="text-sm text-foreground leading-relaxed">
                {risk.treatment_plan || 'No treatment plan documented.'}
              </p>
            </div>

            {/* Controls */}
            {risk.controls.length > 0 && (
              <>
                <Separator />
                <div className="space-y-3">
                  <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
                    Controls
                  </h3>
                  <div className="flex flex-wrap gap-1.5">
                    {risk.controls.map((control) => (
                      <Badge key={control} variant="secondary" className="text-xs">
                        <Shield className="mr-1 h-3 w-3" />
                        {control}
                      </Badge>
                    ))}
                  </div>
                </div>
              </>
            )}

            {/* Business Services */}
            {risk.business_services.length > 0 && (
              <>
                <Separator />
                <div className="space-y-3">
                  <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
                    Business Services
                  </h3>
                  <div className="flex flex-wrap gap-1.5">
                    {risk.business_services.map((service) => (
                      <Badge key={service} variant="outline" className="text-xs">
                        <FileText className="mr-1 h-3 w-3" />
                        {service}
                      </Badge>
                    ))}
                  </div>
                </div>
              </>
            )}

            {/* Tags */}
            {risk.tags.length > 0 && (
              <>
                <Separator />
                <div className="space-y-3">
                  <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
                    Tags
                  </h3>
                  <div className="flex flex-wrap gap-1.5">
                    {risk.tags.map((tag) => (
                      <Badge key={tag} variant="outline" className="text-xs">
                        <Tag className="mr-1 h-3 w-3" />
                        {tag}
                      </Badge>
                    ))}
                  </div>
                </div>
              </>
            )}

            {/* Acceptance Info */}
            {risk.status === 'accepted' && risk.acceptance_rationale && (
              <>
                <Separator />
                <div className="space-y-3">
                  <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
                    Risk Acceptance
                  </h3>
                  <div className="rounded-xl border border-green-200 bg-green-50 p-4 space-y-2">
                    <div className="flex items-center gap-2 text-sm">
                      <CheckCircle className="h-4 w-4 text-green-600" />
                      <span className="font-medium text-green-800">Risk Accepted</span>
                    </div>
                    <p className="text-sm text-green-700">{risk.acceptance_rationale}</p>
                    {risk.acceptance_approved_by_name && (
                      <p className="text-xs text-green-600">
                        Approved by: {risk.acceptance_approved_by_name}
                      </p>
                    )}
                    {risk.acceptance_expiry && (
                      <p className="text-xs text-green-600">
                        Expires: {formatDate(risk.acceptance_expiry)}
                      </p>
                    )}
                  </div>
                </div>
              </>
            )}
          </>
        )}
      </div>
    </DetailPanel>
  );
}
