'use client';

import { useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import {
  ArrowLeft,
  CheckCircle2,
  XCircle,
  Lock,
  Unlock,
  Globe,
  Shield,
  Database,
  FileWarning,
  History,
  RefreshCw,
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { DataAsset } from '@/types/cyber';

type TabId = 'overview' | 'access' | 'compliance' | 'findings' | 'history';

const CLASSIFICATION_COLORS: Record<string, string> = {
  public: 'bg-green-100 text-green-700',
  internal: 'bg-blue-100 text-blue-700',
  confidential: 'bg-amber-100 text-amber-800',
  restricted: 'bg-red-100 text-red-700',
  top_secret: 'bg-purple-100 text-purple-700',
};

const TABS: { id: TabId; label: string }[] = [
  { id: 'overview', label: 'Overview' },
  { id: 'access', label: 'Access' },
  { id: 'compliance', label: 'Compliance' },
  { id: 'findings', label: 'Findings' },
  { id: 'history', label: 'History' },
];

export default function DataAssetDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const id = params?.id ?? '';
  const [activeTab, setActiveTab] = useState<TabId>('overview');

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['cyber-dspm-asset', id],
    queryFn: () => apiGet<{ data: DataAsset }>(API_ENDPOINTS.CYBER_DSPM_DATA_ASSETS + '/' + id),
  });

  const asset = data?.data;

  function ScoreDisplay({ label, score, invert = false }: { label: string; score: number; invert?: boolean }) {
    const color = invert
      ? score <= 30 ? 'text-green-600' : score <= 60 ? 'text-amber-600' : 'text-red-600'
      : score >= 80 ? 'text-green-600' : score >= 60 ? 'text-amber-600' : 'text-red-600';
    return (
      <div className="text-center">
        <p className="text-xs text-muted-foreground">{label}</p>
        <p className={`text-2xl font-bold tabular-nums ${color}`}>{score.toFixed(0)}</p>
      </div>
    );
  }

  function renderOverview() {
    if (!asset) return null;
    return (
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Classification & Sensitivity</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Classification</span>
              <span className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${CLASSIFICATION_COLORS[asset.data_classification] ?? 'bg-muted text-muted-foreground'}`}>
                {asset.data_classification.replace(/_/g, ' ')}
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Sensitivity Score</span>
              <span className="text-sm font-medium tabular-nums">{asset.sensitivity_score.toFixed(0)}/100</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Contains PII</span>
              <span className="flex items-center gap-1 text-sm">
                {asset.contains_pii ? (
                  <><CheckCircle2 className="h-4 w-4 text-amber-500" /> Yes ({asset.pii_column_count} columns)</>
                ) : (
                  <><XCircle className="h-4 w-4 text-green-500" /> No</>
                )}
              </span>
            </div>
            {asset.estimated_record_count != null && (
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground">Estimated Records</span>
                <span className="text-sm font-medium tabular-nums">{asset.estimated_record_count.toLocaleString()}</span>
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Encryption Status</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Encrypted at Rest</span>
              <span className="flex items-center gap-1.5 text-sm">
                {asset.encrypted_at_rest ? (
                  <><Lock className="h-4 w-4 text-green-500" /> Yes</>
                ) : (
                  <><Unlock className="h-4 w-4 text-red-500" /> No</>
                )}
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Encrypted in Transit</span>
              <span className="flex items-center gap-1.5 text-sm">
                {asset.encrypted_in_transit ? (
                  <><Lock className="h-4 w-4 text-green-500" /> Yes</>
                ) : (
                  <><Unlock className="h-4 w-4 text-red-500" /> No</>
                )}
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Network Exposure</span>
              <span className={`flex items-center gap-1.5 text-sm ${asset.network_exposure === 'internet_facing' ? 'text-red-600' : 'text-muted-foreground'}`}>
                {asset.network_exposure === 'internet_facing' && <Globe className="h-4 w-4" />}
                <span className="capitalize">{(asset.network_exposure ?? 'unknown').replace(/_/g, ' ')}</span>
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Access Control</span>
              <span className="text-sm capitalize">{(asset.access_control_type ?? 'none').replace(/_/g, ' ')}</span>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Operational Status</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Backup Configured</span>
              <span className="flex items-center gap-1.5 text-sm">
                {asset.backup_configured ? (
                  <><CheckCircle2 className="h-4 w-4 text-green-500" /> Yes</>
                ) : (
                  <><XCircle className="h-4 w-4 text-red-500" /> No</>
                )}
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">Audit Logging</span>
              <span className="flex items-center gap-1.5 text-sm">
                {asset.audit_logging ? (
                  <><CheckCircle2 className="h-4 w-4 text-green-500" /> Enabled</>
                ) : (
                  <><XCircle className="h-4 w-4 text-red-500" /> Disabled</>
                )}
              </span>
            </div>
            {asset.last_access_review && (
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground">Last Access Review</span>
                <span className="text-sm">{new Date(asset.last_access_review).toLocaleDateString()}</span>
              </div>
            )}
            {asset.last_scanned_at && (
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground">Last Scanned</span>
                <span className="text-sm">{new Date(asset.last_scanned_at).toLocaleDateString()}</span>
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-sm">PII Types Detected</CardTitle>
          </CardHeader>
          <CardContent>
            {asset.pii_types.length === 0 ? (
              <p className="text-sm text-muted-foreground">No PII types detected</p>
            ) : (
              <div className="flex flex-wrap gap-2">
                {asset.pii_types.map((pii) => (
                  <Badge key={pii} variant="outline" className="text-xs">
                    {pii.replace(/_/g, ' ')}
                  </Badge>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    );
  }

  function renderAccess() {
    return (
      <Card>
        <CardContent className="flex flex-col items-center justify-center py-12 text-center">
          <Shield className="mb-4 h-10 w-10 text-muted-foreground" />
          <h3 className="mb-2 text-base font-medium">Access Details</h3>
          <p className="mb-4 max-w-md text-sm text-muted-foreground">
            Detailed access intelligence including identity mappings, overprivileged accounts,
            and blast radius analysis is available in the Access Intelligence module.
          </p>
          <Button variant="outline" size="sm" onClick={() => router.push('/cyber/dspm/access')}>
            Open Access Intelligence
          </Button>
        </CardContent>
      </Card>
    );
  }

  function renderCompliance() {
    if (!asset) return null;
    const tags = asset.metadata?.compliance_tags ?? [];
    if (tags.length === 0) {
      return (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12 text-center">
            <Shield className="mb-4 h-10 w-10 text-muted-foreground" />
            <h3 className="mb-2 text-base font-medium">No Compliance Tags</h3>
            <p className="text-sm text-muted-foreground">
              This asset has no compliance framework tags attached yet.
            </p>
          </CardContent>
        </Card>
      );
    }

    const grouped: Record<string, typeof tags> = {};
    for (const tag of tags) {
      const fw = tag.framework.toUpperCase();
      if (!grouped[fw]) grouped[fw] = [];
      grouped[fw].push(tag);
    }

    return (
      <div className="space-y-4">
        {Object.entries(grouped).map(([framework, frameworkTags]) => (
          <Card key={framework}>
            <CardHeader>
              <CardTitle className="text-sm">{framework}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {frameworkTags.map((tag) => (
                  <div key={`${tag.framework}-${tag.article}`} className="flex items-start justify-between rounded-lg border p-3">
                    <div className="space-y-1">
                      <p className="text-sm font-medium">{tag.article}</p>
                      <p className="text-xs text-muted-foreground">{tag.requirement}</p>
                      <Badge variant="outline" className="text-xs">{tag.category}</Badge>
                    </div>
                    <div className="flex flex-col items-end gap-1">
                      <SeverityIndicator severity={tag.severity} size="sm" />
                      <span className="text-xs text-muted-foreground">{tag.impact}</span>
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    );
  }

  function renderFindings() {
    if (!asset) return null;
    const findings = asset.posture_findings;
    if (findings.length === 0) {
      return (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12 text-center">
            <CheckCircle2 className="mb-4 h-10 w-10 text-green-500" />
            <h3 className="mb-2 text-base font-medium">No Findings</h3>
            <p className="text-sm text-muted-foreground">
              This asset has a clean posture with no active findings.
            </p>
          </CardContent>
        </Card>
      );
    }

    return (
      <div className="space-y-3">
        {findings.map((finding, idx) => (
          <Card key={`${finding.control}-${idx}`}>
            <CardContent className="flex items-start justify-between p-4">
              <div className="space-y-1">
                <div className="flex items-center gap-2">
                  <FileWarning className="h-4 w-4 text-muted-foreground" />
                  <p className="text-sm font-medium">{finding.control}</p>
                </div>
                <p className="text-xs text-muted-foreground">{finding.description}</p>
                <p className="text-xs text-blue-600">{finding.guidance}</p>
              </div>
              <SeverityIndicator severity={finding.severity} size="sm" />
            </CardContent>
          </Card>
        ))}
      </div>
    );
  }

  function renderHistory() {
    return (
      <Card>
        <CardContent className="flex flex-col items-center justify-center py-12 text-center">
          <History className="mb-4 h-10 w-10 text-muted-foreground" />
          <h3 className="mb-2 text-base font-medium">Remediation History</h3>
          <p className="mb-4 max-w-md text-sm text-muted-foreground">
            View all past and active remediation actions taken on this data asset.
          </p>
          <Button variant="outline" size="sm" onClick={() => router.push('/cyber/dspm/remediations')}>
            View Remediations
          </Button>
        </CardContent>
      </Card>
    );
  }

  function renderTabContent() {
    switch (activeTab) {
      case 'overview': return renderOverview();
      case 'access': return renderAccess();
      case 'compliance': return renderCompliance();
      case 'findings': return renderFindings();
      case 'history': return renderHistory();
      default: return null;
    }
  }

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        {isLoading ? (
          <>
            <div className="h-8 w-64 animate-pulse rounded bg-muted" />
            <LoadingSkeleton variant="card" count={2} />
          </>
        ) : error || !asset ? (
          <ErrorState message="Failed to load data asset details" onRetry={() => void refetch()} />
        ) : (
          <>
            <PageHeader
              title={
                <div className="flex items-center gap-3">
                  <button
                    onClick={() => router.push('/cyber/dspm/assets')}
                    className="flex h-8 w-8 items-center justify-center rounded-full border bg-background text-muted-foreground shadow-sm transition-colors hover:bg-accent"
                  >
                    <ArrowLeft className="h-4 w-4" />
                  </button>
                  <span className="truncate">{asset.asset_name}</span>
                </div>
              }
              description={
                <div className="flex flex-wrap items-center gap-3 pl-11">
                  <Badge variant="outline" className="capitalize">{asset.asset_type.replace(/_/g, ' ')}</Badge>
                  <span className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${CLASSIFICATION_COLORS[asset.data_classification] ?? 'bg-muted text-muted-foreground'}`}>
                    {asset.data_classification.replace(/_/g, ' ')}
                  </span>
                  {asset.database_type && (
                    <Badge variant="secondary" className="text-xs">
                      <Database className="mr-1 h-3 w-3" />
                      {asset.database_type}
                    </Badge>
                  )}
                </div>
              }
              actions={
                <Button variant="outline" size="sm" onClick={() => void refetch()}>
                  <RefreshCw className="mr-1.5 h-4 w-4" />
                  Refresh
                </Button>
              }
            />

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <ScoreDisplay label="Posture Score" score={asset.posture_score} />
              <ScoreDisplay label="Risk Score" score={asset.risk_score} invert />
              <ScoreDisplay label="Sensitivity" score={asset.sensitivity_score} invert />
              <div className="text-center">
                <p className="text-xs text-muted-foreground">Findings</p>
                <p className={`text-2xl font-bold tabular-nums ${asset.posture_findings.length > 0 ? 'text-orange-600' : 'text-green-600'}`}>
                  {asset.posture_findings.length}
                </p>
              </div>
            </div>

            <div className="border-b">
              <nav className="-mb-px flex gap-4 overflow-x-auto">
                {TABS.map((tab) => (
                  <button
                    key={tab.id}
                    onClick={() => setActiveTab(tab.id)}
                    className={`whitespace-nowrap border-b-2 px-1 pb-3 text-sm font-medium transition-colors ${
                      activeTab === tab.id
                        ? 'border-primary text-primary'
                        : 'border-transparent text-muted-foreground hover:border-muted-foreground/30 hover:text-foreground'
                    }`}
                  >
                    {tab.label}
                  </button>
                ))}
              </nav>
            </div>

            {renderTabContent()}
          </>
        )}
      </div>
    </PermissionRedirect>
  );
}
