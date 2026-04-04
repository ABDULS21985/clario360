'use client';

import { useState } from 'react';
import { toast } from 'sonner';
import {
  Shield,
  Calendar,
  User,
  Mail,
  Database,
  Server,
  AlertTriangle,
  CheckCircle,
  Clock,
  Edit,
  Save,
  X,
} from 'lucide-react';
import { DetailPanel } from '@/components/shared/detail-panel';
import { StatusBadge } from '@/components/shared/status-badge';
import { SeverityIndicator, type Severity } from '@/components/shared/severity-indicator';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Separator } from '@/components/ui/separator';
import { Progress } from '@/components/ui/progress';
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
import { vendorStatusConfig } from '@/lib/status-configs';
import type { VCISOVendor, VendorRiskTier, VendorStatus } from '@/types/cyber';

interface VendorDetailPanelProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  vendor: VCISOVendor;
  onUpdated: () => void;
}

const RISK_TIER_OPTIONS: { label: string; value: VendorRiskTier }[] = [
  { label: 'Critical', value: 'critical' },
  { label: 'High', value: 'high' },
  { label: 'Medium', value: 'medium' },
  { label: 'Low', value: 'low' },
];

const STATUS_OPTIONS: { label: string; value: VendorStatus }[] = [
  { label: 'Active', value: 'active' },
  { label: 'Onboarding', value: 'onboarding' },
  { label: 'Under Review', value: 'under_review' },
  { label: 'Offboarding', value: 'offboarding' },
  { label: 'Terminated', value: 'terminated' },
];

function riskScoreColor(score: number): string {
  if (score >= 80) return 'text-red-600';
  if (score >= 60) return 'text-orange-600';
  if (score >= 40) return 'text-amber-600';
  return 'text-green-600';
}

function riskScoreBg(score: number): string {
  if (score >= 80) return 'bg-red-100 text-red-700';
  if (score >= 60) return 'bg-orange-100 text-orange-700';
  if (score >= 40) return 'bg-amber-100 text-amber-700';
  return 'bg-green-100 text-green-700';
}

export function VendorDetailPanel({
  open,
  onOpenChange,
  vendor,
  onUpdated,
}: VendorDetailPanelProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [editForm, setEditForm] = useState({
    name: vendor.name,
    category: vendor.category,
    risk_tier: vendor.risk_tier,
    status: vendor.status,
    contact_name: vendor.contact_name ?? '',
    contact_email: vendor.contact_email ?? '',
    services_provided: vendor.services_provided.join(', '),
    data_shared: vendor.data_shared.join(', '),
    next_review_date: vendor.next_review_date ? vendor.next_review_date.split('T')[0] : '',
  });

  const updateMutation = useApiMutation<VCISOVendor, Record<string, unknown>>(
    'put',
    `${API_ENDPOINTS.CYBER_VCISO_VENDORS}/${vendor.id}`,
    {
      successMessage: 'Vendor updated successfully',
      invalidateKeys: ['vciso-vendors'],
      onSuccess: () => {
        setIsEditing(false);
        onUpdated();
      },
    },
  );

  const handleSave = () => {
    if (!editForm.name.trim()) {
      toast.error('Name is required');
      return;
    }

    updateMutation.mutate({
      name: editForm.name.trim(),
      category: editForm.category.trim(),
      risk_tier: editForm.risk_tier,
      status: editForm.status,
      contact_name: editForm.contact_name.trim() || undefined,
      contact_email: editForm.contact_email.trim() || undefined,
      services_provided: editForm.services_provided
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean),
      data_shared: editForm.data_shared
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean),
      next_review_date: editForm.next_review_date || undefined,
    });
  };

  const handleCancel = () => {
    setIsEditing(false);
    setEditForm({
      name: vendor.name,
      category: vendor.category,
      risk_tier: vendor.risk_tier,
      status: vendor.status,
      contact_name: vendor.contact_name ?? '',
      contact_email: vendor.contact_email ?? '',
      services_provided: vendor.services_provided.join(', '),
      data_shared: vendor.data_shared.join(', '),
      next_review_date: vendor.next_review_date ? vendor.next_review_date.split('T')[0] : '',
    });
  };

  const handleOpenChange = (o: boolean) => {
    if (!o) setIsEditing(false);
    onOpenChange(o);
  };

  const controlsPercent =
    vendor.controls_total > 0
      ? Math.round((vendor.controls_met / vendor.controls_total) * 100)
      : 0;

  return (
    <DetailPanel
      open={open}
      onOpenChange={handleOpenChange}
      title={isEditing ? 'Edit Vendor' : vendor.name}
      description={isEditing ? 'Update vendor details below' : vendor.category}
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
              <Label htmlFor="edit-vendor-name">Name</Label>
              <Input
                id="edit-vendor-name"
                value={editForm.name}
                onChange={(e) => setEditForm((f) => ({ ...f, name: e.target.value }))}
                placeholder="Vendor name"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="edit-vendor-category">Category</Label>
              <Input
                id="edit-vendor-category"
                value={editForm.category}
                onChange={(e) => setEditForm((f) => ({ ...f, category: e.target.value }))}
                placeholder="e.g. Cloud Infrastructure"
              />
            </div>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label>Risk Tier</Label>
                <Select
                  value={editForm.risk_tier}
                  onValueChange={(v) => setEditForm((f) => ({ ...f, risk_tier: v as VendorRiskTier }))}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {RISK_TIER_OPTIONS.map((opt) => (
                      <SelectItem key={opt.value} value={opt.value}>
                        {opt.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>Status</Label>
                <Select
                  value={editForm.status}
                  onValueChange={(v) => setEditForm((f) => ({ ...f, status: v as VendorStatus }))}
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
            </div>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="edit-contact-name">Contact Name</Label>
                <Input
                  id="edit-contact-name"
                  value={editForm.contact_name}
                  onChange={(e) => setEditForm((f) => ({ ...f, contact_name: e.target.value }))}
                  placeholder="Contact person"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="edit-contact-email">Contact Email</Label>
                <Input
                  id="edit-contact-email"
                  type="email"
                  value={editForm.contact_email}
                  onChange={(e) => setEditForm((f) => ({ ...f, contact_email: e.target.value }))}
                  placeholder="vendor@example.com"
                />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="edit-next-review">Next Review Date</Label>
              <Input
                id="edit-next-review"
                type="date"
                value={editForm.next_review_date}
                onChange={(e) => setEditForm((f) => ({ ...f, next_review_date: e.target.value }))}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="edit-services">Services Provided (comma-separated)</Label>
              <Input
                id="edit-services"
                value={editForm.services_provided}
                onChange={(e) => setEditForm((f) => ({ ...f, services_provided: e.target.value }))}
                placeholder="Cloud Hosting, CDN, DNS"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="edit-data">Data Shared (comma-separated)</Label>
              <Input
                id="edit-data"
                value={editForm.data_shared}
                onChange={(e) => setEditForm((f) => ({ ...f, data_shared: e.target.value }))}
                placeholder="PII, Financial, Logs"
              />
            </div>
          </div>
        ) : (
          /* ── Read-only view ───────────────────────────────────── */
          <>
            {/* Overview */}
            <div className="space-y-3">
              <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
                Overview
              </h3>
              <div className="flex flex-wrap gap-2">
                <StatusBadge status={vendor.status} config={vendorStatusConfig} />
                <SeverityIndicator severity={vendor.risk_tier as Severity} showLabel />
                <Badge variant="outline">{vendor.category}</Badge>
              </div>
            </div>

            <Separator />

            {/* Risk Score & Controls */}
            <div className="space-y-3">
              <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
                Risk Assessment
              </h3>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div className="rounded-xl border p-4 text-center">
                  <p className="text-xs text-muted-foreground mb-1">Risk Score</p>
                  <p className={cn('text-2xl font-bold', riskScoreColor(vendor.risk_score))}>
                    {vendor.risk_score}
                  </p>
                </div>
                <div className="rounded-xl border p-4 text-center">
                  <p className="text-xs text-muted-foreground mb-1">Open Findings</p>
                  <p className={cn('text-2xl font-bold', vendor.open_findings > 0 ? 'text-red-600' : 'text-green-600')}>
                    {vendor.open_findings}
                  </p>
                </div>
              </div>

              <div className="rounded-xl border p-4 space-y-2">
                <div className="flex items-center justify-between text-sm">
                  <span className="text-muted-foreground">Controls Coverage</span>
                  <span className="font-medium">
                    {vendor.controls_met}/{vendor.controls_total} ({controlsPercent}%)
                  </span>
                </div>
                <Progress value={controlsPercent} className="h-2" />
              </div>
            </div>

            <Separator />

            {/* Contact Details */}
            <div className="space-y-3">
              <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
                Contact Details
              </h3>
              <div className="space-y-2">
                <div className="flex items-center gap-2 text-sm">
                  <User className="h-4 w-4 text-muted-foreground" />
                  <span className="text-muted-foreground">Contact:</span>
                  <span className="font-medium">{vendor.contact_name || 'Not provided'}</span>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <Mail className="h-4 w-4 text-muted-foreground" />
                  <span className="text-muted-foreground">Email:</span>
                  <span className="font-medium">{vendor.contact_email || 'Not provided'}</span>
                </div>
              </div>
            </div>

            <Separator />

            {/* Timeline */}
            <div className="space-y-3">
              <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
                Timeline
              </h3>
              <div className="space-y-2">
                <div className="flex items-center gap-2 text-sm">
                  <Calendar className="h-4 w-4 text-muted-foreground" />
                  <span className="text-muted-foreground">Last Assessment:</span>
                  <span className="font-medium">
                    {vendor.last_assessment_date ? formatDate(vendor.last_assessment_date) : 'Never'}
                  </span>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <Calendar className="h-4 w-4 text-muted-foreground" />
                  <span className="text-muted-foreground">Next Review:</span>
                  <span className="font-medium">{formatDate(vendor.next_review_date)}</span>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <Clock className="h-4 w-4 text-muted-foreground" />
                  <span className="text-muted-foreground">Created:</span>
                  <span className="font-medium">{formatDate(vendor.created_at)}</span>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  <Clock className="h-4 w-4 text-muted-foreground" />
                  <span className="text-muted-foreground">Updated:</span>
                  <span className="font-medium">{formatDate(vendor.updated_at)}</span>
                </div>
              </div>
            </div>

            {/* Services Provided */}
            {vendor.services_provided.length > 0 && (
              <>
                <Separator />
                <div className="space-y-3">
                  <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
                    Services Provided
                  </h3>
                  <div className="flex flex-wrap gap-1.5">
                    {vendor.services_provided.map((service) => (
                      <Badge key={service} variant="secondary" className="text-xs">
                        <Server className="mr-1 h-3 w-3" />
                        {service}
                      </Badge>
                    ))}
                  </div>
                </div>
              </>
            )}

            {/* Data Shared */}
            {vendor.data_shared.length > 0 && (
              <>
                <Separator />
                <div className="space-y-3">
                  <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
                    Data Shared
                  </h3>
                  <div className="flex flex-wrap gap-1.5">
                    {vendor.data_shared.map((data) => (
                      <Badge key={data} variant="outline" className="text-xs">
                        <Database className="mr-1 h-3 w-3" />
                        {data}
                      </Badge>
                    ))}
                  </div>
                </div>
              </>
            )}

            {/* Compliance Frameworks */}
            {vendor.compliance_frameworks.length > 0 && (
              <>
                <Separator />
                <div className="space-y-3">
                  <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider">
                    Compliance Frameworks
                  </h3>
                  <div className="flex flex-wrap gap-1.5">
                    {vendor.compliance_frameworks.map((framework) => (
                      <Badge key={framework} variant="secondary" className="text-xs">
                        <Shield className="mr-1 h-3 w-3" />
                        {framework}
                      </Badge>
                    ))}
                  </div>
                </div>
              </>
            )}

            {/* Open findings warning */}
            {vendor.open_findings > 0 && (
              <>
                <Separator />
                <div className="rounded-xl border border-red-200 bg-red-50 p-4 space-y-2">
                  <div className="flex items-center gap-2 text-sm">
                    <AlertTriangle className="h-4 w-4 text-red-600" />
                    <span className="font-medium text-red-800">
                      {vendor.open_findings} Open Finding{vendor.open_findings !== 1 ? 's' : ''}
                    </span>
                  </div>
                  <p className="text-sm text-red-700">
                    This vendor has unresolved findings that require attention.
                  </p>
                </div>
              </>
            )}

            {/* All clear indicator */}
            {vendor.open_findings === 0 && controlsPercent === 100 && (
              <>
                <Separator />
                <div className="rounded-xl border border-green-200 bg-green-50 p-4">
                  <div className="flex items-center gap-2 text-sm">
                    <CheckCircle className="h-4 w-4 text-green-600" />
                    <span className="font-medium text-green-800">Fully Compliant</span>
                  </div>
                  <p className="text-sm text-green-700 mt-1">
                    All controls are met and no open findings.
                  </p>
                </div>
              </>
            )}
          </>
        )}
      </div>
    </DetailPanel>
  );
}
