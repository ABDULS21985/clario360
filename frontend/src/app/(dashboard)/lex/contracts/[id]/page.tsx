'use client';

import { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  ArrowUpRight,
  FileSearch,
  FileUp,
  GitBranch,
  PencilLine,
  PlayCircle,
  RefreshCw,
  ShieldCheck,
  Trash2,
} from 'lucide-react';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { EmptyState } from '@/components/common/empty-state';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { RelativeTime } from '@/components/shared/relative-time';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { StatusBadge } from '@/components/shared/status-badge';
import { SectionCard } from '@/components/suites/section-card';
import { Badge } from '@/components/ui/badge';
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
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Textarea } from '@/components/ui/textarea';
import { useAuth } from '@/hooks/use-auth';
import { enterpriseApi, userDisplayName } from '@/lib/enterprise';
import {
  downloadBlob,
  formatBytes,
  formatDateTime,
  formatNumber,
  titleCase,
} from '@/lib/format';
import {
  clauseReviewStatusConfig,
  contractStatusConfig,
} from '@/lib/status-configs';
import { showApiError, showSuccess } from '@/lib/toast';
import type {
  LexClause,
  LexContractRecord,
  LexContractStatus,
  LexContractVersion,
  UserDirectoryEntry,
} from '@/types/suites';
import { ContractFormDialog } from '../_components/contract-form-dialog';

const STATUS_TRANSITIONS: Record<LexContractStatus, LexContractStatus[]> = {
  draft: ['internal_review', 'cancelled'],
  internal_review: ['legal_review', 'draft'],
  legal_review: ['negotiation', 'internal_review', 'draft'],
  negotiation: ['pending_signature', 'cancelled', 'draft'],
  pending_signature: ['active', 'cancelled'],
  active: ['suspended', 'terminated', 'expired', 'renewed'],
  suspended: ['active', 'terminated'],
  expired: ['renewed'],
  terminated: [],
  renewed: [],
  cancelled: [],
};

type ContractTab = 'overview' | 'analysis' | 'versions' | 'workflow';

type ClauseReviewDraft = {
  notes: string;
  status: LexClause['review_status'];
};

type RenewDraft = {
  changeSummary: string;
  newEffectiveDate: string;
  newExpiryDate: string;
  newValue: string;
};

type ReviewDraft = {
  approverRole: string;
  approverUserId: string;
  description: string;
  slaHours: string;
};

export default function LexContractDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const queryClient = useQueryClient();
  const { hasPermission } = useAuth();
  const contractId = params?.id ?? '';
  const canWrite = hasPermission('lex:write');

  const [activeTab, setActiveTab] = useState<ContractTab>('overview');
  const [analysisMessage, setAnalysisMessage] = useState<string | null>(null);
  const [complianceResult, setComplianceResult] = useState<{
    alerts_created: number;
    calculated_at: string;
    score: number;
  } | null>(null);
  const [editOpen, setEditOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [statusOpen, setStatusOpen] = useState(false);
  const [uploadOpen, setUploadOpen] = useState(false);
  const [reviewOpen, setReviewOpen] = useState(false);
  const [renewOpen, setRenewOpen] = useState(false);
  const [clauseReviewTarget, setClauseReviewTarget] = useState<LexClause | null>(null);

  const contractQuery = useQuery({
    queryKey: ['lex-contract', contractId],
    queryFn: () => enterpriseApi.lex.getContract(contractId),
    enabled: Boolean(contractId),
  });

  const versionsQuery = useQuery({
    queryKey: ['lex-contract-versions', contractId],
    queryFn: () => enterpriseApi.lex.listContractVersions(contractId),
    enabled: Boolean(contractId),
  });

  const usersQuery = useQuery({
    queryKey: ['enterprise-users', 'lex-contract-review', contractId],
    queryFn: () => enterpriseApi.users.list({ page: 1, per_page: 200, order: 'asc' }),
    enabled: canWrite && reviewOpen,
  });

  const refreshContract = async () => {
    await Promise.all([
      contractQuery.refetch(),
      versionsQuery.refetch(),
      queryClient.invalidateQueries({ queryKey: ['lex-contracts'] }),
      queryClient.invalidateQueries({ queryKey: ['lex-overview'] }),
    ]);
  };

  const analyzeMutation = useMutation({
    mutationFn: () => enterpriseApi.lex.analyzeContract(contractId),
    onSuccess: async (analysis) => {
      setAnalysisMessage(
        `Analysis completed with ${analysis.key_findings.length} findings and ${analysis.compliance_flags.length} compliance flags.`,
      );
      showSuccess('Contract analyzed.', 'The latest clause and risk analysis is now available.');
      await refreshContract();
      setActiveTab('analysis');
    },
    onError: showApiError,
  });

  const complianceMutation = useMutation({
    mutationFn: () => enterpriseApi.lex.runCompliance({ contract_ids: [contractId] }),
    onSuccess: async (result) => {
      setComplianceResult({
        alerts_created: result.alerts_created,
        calculated_at: result.calculated_at,
        score: result.score,
      });
      showSuccess(
        'Compliance checks completed.',
        `${result.alerts_created} alert${result.alerts_created === 1 ? '' : 's'} created for this contract.`,
      );
      await refreshContract();
      setActiveTab('overview');
    },
    onError: showApiError,
  });

  const statusMutation = useMutation({
    mutationFn: (nextStatus: LexContractStatus) =>
      enterpriseApi.lex.updateContractStatus(contractId, { status: nextStatus }),
    onSuccess: async () => {
      showSuccess('Status updated.', 'The contract lifecycle state has been changed.');
      await refreshContract();
      setStatusOpen(false);
    },
    onError: showApiError,
  });

  const renewMutation = useMutation({
    mutationFn: (payload: Record<string, unknown>) =>
      enterpriseApi.lex.renewContract(contractId, payload),
    onSuccess: async () => {
      showSuccess('Contract renewed.', 'A renewed contract record has been created.');
      await refreshContract();
      setRenewOpen(false);
    },
    onError: showApiError,
  });

  const reviewMutation = useMutation({
    mutationFn: (payload: Record<string, unknown>) =>
      enterpriseApi.lex.startContractReview(contractId, payload),
    onSuccess: async () => {
      showSuccess('Review started.', 'A workflow instance now tracks the contract review.');
      await refreshContract();
      setReviewOpen(false);
      setActiveTab('workflow');
    },
    onError: showApiError,
  });

  const deleteMutation = useMutation({
    mutationFn: () => enterpriseApi.lex.deleteContract(contractId),
    onSuccess: async () => {
      showSuccess('Contract deleted.', 'The contract has been removed from the active portfolio.');
      await queryClient.invalidateQueries({ queryKey: ['lex-contracts'] });
      router.push('/lex/contracts');
    },
    onError: showApiError,
  });

  const clauseReviewMutation = useMutation({
    mutationFn: ({ clauseId, notes, status }: { clauseId: string; notes: string; status: LexClause['review_status'] }) =>
      enterpriseApi.lex.updateClauseReview(contractId, clauseId, { status, notes }),
    onSuccess: async () => {
      showSuccess('Clause review saved.', 'The clause review decision is now attached to the contract.');
      await refreshContract();
      setClauseReviewTarget(null);
    },
    onError: showApiError,
  });

  if (contractQuery.isLoading) {
    return (
      <PermissionRedirect permission="lex:read">
        <div className="space-y-6">
          <PageHeader
            title="Contract"
            description="Loading contract lifecycle, analysis, and workflow context."
          />
          <LoadingSkeleton variant="card" count={4} />
        </div>
      </PermissionRedirect>
    );
  }

  if (contractQuery.isError || !contractQuery.data) {
    return (
      <PermissionRedirect permission="lex:read">
        <div className="space-y-6">
          <PageHeader title="Contract" description="Legal contract lifecycle detail." />
          <ErrorState
            message="Failed to load contract details."
            onRetry={() => void contractQuery.refetch()}
          />
        </div>
      </PermissionRedirect>
    );
  }

  const detail = contractQuery.data;
  const contract = detail.contract;
  const clauses = detail.clauses;
  const latestAnalysis = detail.latest_analysis ?? null;
  const versions = versionsQuery.data ?? [];
  const allowedStatuses = STATUS_TRANSITIONS[contract.status] ?? [];
  const latestVersion = versions[0] ?? null;
  const users = usersQuery.data?.data ?? [];

  return (
    <PermissionRedirect permission="lex:read">
      <div className="space-y-6">
        <PageHeader
          title={contract.title}
          description={contract.description || 'Legal contract lifecycle detail.'}
          actions={
            <div className="flex flex-wrap items-center gap-2">
              {canWrite ? (
                <Button variant="outline" onClick={() => setEditOpen(true)}>
                  <PencilLine className="mr-1.5 h-3.5 w-3.5" />
                  Edit
                </Button>
              ) : null}
              <Button
                variant="outline"
                onClick={() => void analyzeMutation.mutate()}
                disabled={analyzeMutation.isPending}
              >
                {analyzeMutation.isPending ? (
                  <RefreshCw className="mr-1.5 h-3.5 w-3.5 animate-spin" />
                ) : (
                  <FileSearch className="mr-1.5 h-3.5 w-3.5" />
                )}
                Analyze
              </Button>
              <Button
                onClick={() => void complianceMutation.mutate()}
                disabled={complianceMutation.isPending}
              >
                {complianceMutation.isPending ? (
                  <RefreshCw className="mr-1.5 h-3.5 w-3.5 animate-spin" />
                ) : (
                  <ShieldCheck className="mr-1.5 h-3.5 w-3.5" />
                )}
                Run Compliance
              </Button>
            </div>
          }
        />

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
          <MetricCard
            label="Status"
            value={
              <StatusBadge status={contract.status} config={contractStatusConfig} />
            }
          />
          <MetricCard
            label="Risk"
            value={
              <div className="flex items-center gap-2">
                <SeverityIndicator severity={normalizeRiskSeverity(contract.risk_level)} size="sm" />
                <span>{titleCase(contract.risk_level)}</span>
              </div>
            }
            helper={contract.risk_score != null ? `Score ${formatNumber(contract.risk_score)}` : 'No score yet'}
          />
          <MetricCard
            label="Version"
            value={`v${contract.current_version}`}
            helper={`${detail.version_count} recorded version${detail.version_count === 1 ? '' : 's'}`}
          />
          <MetricCard
            label="Workflow"
            value={contract.workflow_instance_id ? 'Active review' : 'No workflow'}
            helper={
              contract.workflow_instance_id
                ? `Instance ${contract.workflow_instance_id.slice(0, 8)}`
                : 'Review not started'
            }
          />
        </div>

        {analysisMessage ? (
          <div className="rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-900">
            {analysisMessage}
          </div>
        ) : null}

        <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as ContractTab)}>
          <TabsList className="w-full justify-start">
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="analysis">Analysis &amp; Clauses</TabsTrigger>
            <TabsTrigger value="versions">Versions</TabsTrigger>
            <TabsTrigger value="workflow">Workflow</TabsTrigger>
          </TabsList>

          <TabsContent value="overview" className="space-y-4">
            <div className="grid grid-cols-1 gap-4 xl:grid-cols-[1.15fr_0.85fr]">
              <SectionCard title="Contract Metadata" description="Core contract state, dates, and legal ownership.">
                <div className="space-y-3">
                  <MetadataRow label="Contract number" value={contract.contract_number ?? 'Auto-generated'} />
                  <MetadataRow label="Type" value={titleCase(contract.type)} />
                  <MetadataRow label="Owner" value={contract.owner_name} />
                  <MetadataRow label="Legal reviewer" value={contract.legal_reviewer_name ?? 'Unassigned'} />
                  <MetadataRow label="Department" value={contract.department ?? 'Not set'} />
                  <MetadataRow label="Effective date" value={formatOptionalDate(contract.effective_date)} />
                  <MetadataRow label="Expiry date" value={formatOptionalDate(contract.expiry_date)} />
                  <MetadataRow label="Renewal date" value={formatOptionalDate(contract.renewal_date)} />
                  <MetadataRow label="Payment terms" value={contract.payment_terms ?? 'Not set'} />
                  <MetadataRow
                    label="Tags"
                    value={
                      contract.tags.length > 0 ? (
                        <div className="flex flex-wrap gap-2">
                          {contract.tags.map((tag) => (
                            <Badge key={tag} variant="outline">
                              {tag}
                            </Badge>
                          ))}
                        </div>
                      ) : (
                        'No tags'
                      )
                    }
                  />
                </div>
              </SectionCard>

              <SectionCard title="Lifecycle Actions" description="Real workflow and contract lifecycle mutations backed by lex-service.">
                <div className="grid gap-3">
                  <Button
                    variant="outline"
                    onClick={() => setStatusOpen(true)}
                    disabled={!canWrite || allowedStatuses.length === 0}
                  >
                    Change Status
                  </Button>
                  <Button
                    variant="outline"
                    onClick={() => setReviewOpen(true)}
                    disabled={!canWrite || Boolean(contract.workflow_instance_id)}
                  >
                    <PlayCircle className="mr-1.5 h-3.5 w-3.5" />
                    Start Review Workflow
                  </Button>
                  <Button
                    variant="outline"
                    onClick={() => setRenewOpen(true)}
                    disabled={!canWrite || !['active', 'expired'].includes(contract.status)}
                  >
                    Renew Contract
                  </Button>
                  <Button
                    variant="outline"
                    onClick={() => setUploadOpen(true)}
                    disabled={!canWrite}
                  >
                    <FileUp className="mr-1.5 h-3.5 w-3.5" />
                    Upload New Version
                  </Button>
                  <Button
                    variant="destructive"
                    onClick={() => setDeleteOpen(true)}
                    disabled={!canWrite || deleteMutation.isPending}
                  >
                    <Trash2 className="mr-1.5 h-3.5 w-3.5" />
                    Delete Contract
                  </Button>
                </div>
              </SectionCard>
            </div>

            <div className="grid grid-cols-1 gap-4 xl:grid-cols-[1fr_1fr]">
              <SectionCard title="Parties & Value" description="Commercial parties and financial context.">
                <div className="space-y-3">
                  <MetadataRow label="Party A" value={contract.party_a_name} />
                  <MetadataRow label="Party A entity" value={contract.party_a_entity ?? 'Not set'} />
                  <MetadataRow label="Counterparty" value={contract.party_b_name} />
                  <MetadataRow label="Counterparty entity" value={contract.party_b_entity ?? 'Not set'} />
                  <MetadataRow label="Counterparty contact" value={contract.party_b_contact ?? 'Not set'} />
                  <MetadataRow
                    label="Total value"
                    value={
                      contract.total_value != null
                        ? `${contract.currency} ${formatNumber(contract.total_value)}`
                        : 'Undisclosed'
                    }
                  />
                </div>
              </SectionCard>

              <SectionCard title="Document Context" description="Latest version and workflow linkage for downstream review.">
                <div className="space-y-3">
                  <MetadataRow
                    label="Latest version"
                    value={latestVersion ? `v${latestVersion.version} • ${latestVersion.file_name}` : 'No uploaded versions'}
                  />
                  <MetadataRow
                    label="Latest upload"
                    value={
                      latestVersion ? (
                        <div className="flex items-center gap-2">
                          <span>{formatDateTime(latestVersion.uploaded_at)}</span>
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={() => void downloadVersion(latestVersion)}
                          >
                            Download
                          </Button>
                        </div>
                      ) : (
                        'No file available'
                      )
                    }
                  />
                  <MetadataRow
                    label="Workflow instance"
                    value={
                      contract.workflow_instance_id ? (
                        <Link
                          href={`/workflows/instances/${contract.workflow_instance_id}`}
                          className="inline-flex items-center gap-1 text-primary hover:underline"
                        >
                          {contract.workflow_instance_id}
                          <ArrowUpRight className="h-3.5 w-3.5" />
                        </Link>
                      ) : (
                        'Not linked'
                      )
                    }
                  />
                  <MetadataRow
                    label="Last analyzed"
                    value={contract.last_analyzed_at ? <RelativeTime date={contract.last_analyzed_at} /> : 'Not analyzed'}
                  />
                  <MetadataRow
                    label="Analysis status"
                    value={titleCase(contract.analysis_status)}
                  />
                </div>
              </SectionCard>
            </div>

            {complianceResult ? (
              <SectionCard title="Latest Compliance Run" description="Most recent contract-scoped compliance execution from the live backend.">
                <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
                  <MetricCard label="Score" value={`${formatNumber(complianceResult.score)}%`} />
                  <MetricCard label="Alerts Created" value={String(complianceResult.alerts_created)} />
                  <MetricCard label="Calculated At" value={formatDateTime(complianceResult.calculated_at)} />
                </div>
              </SectionCard>
            ) : null}
          </TabsContent>

          <TabsContent value="analysis" className="space-y-4">
            {latestAnalysis ? (
              <>
                <div className="grid grid-cols-1 gap-4 xl:grid-cols-[1fr_1fr]">
                  <SectionCard title="Risk Summary" description="Latest explainable analysis from lex-service.">
                    <div className="space-y-3">
                      <MetadataRow label="Overall risk" value={titleCase(latestAnalysis.overall_risk)} />
                      <MetadataRow label="Risk score" value={formatNumber(latestAnalysis.risk_score)} />
                      <MetadataRow label="Clause count" value={String(latestAnalysis.clause_count)} />
                      <MetadataRow label="High-risk clauses" value={String(latestAnalysis.high_risk_clause_count)} />
                      <MetadataRow label="Analyzed at" value={formatDateTime(latestAnalysis.analyzed_at)} />
                      <MetadataRow label="Analysis duration" value={`${latestAnalysis.analysis_duration_ms} ms`} />
                    </div>
                  </SectionCard>

                  <SectionCard title="Extracted Parties & Dates" description="Deterministic fields extracted during analysis.">
                    <div className="space-y-4">
                      <div className="space-y-2">
                        <p className="text-sm font-medium">Parties</p>
                        {latestAnalysis.extracted_parties.length > 0 ? (
                          latestAnalysis.extracted_parties.map((party) => (
                            <div key={`${party.name}-${party.role}`} className="rounded-lg border px-3 py-2 text-sm">
                              <div className="font-medium">{party.name}</div>
                              <div className="text-muted-foreground">
                                {party.role} • {party.source}
                              </div>
                            </div>
                          ))
                        ) : (
                          <p className="text-sm text-muted-foreground">No parties extracted.</p>
                        )}
                      </div>

                      <div className="space-y-2">
                        <p className="text-sm font-medium">Dates</p>
                        {latestAnalysis.extracted_dates.length > 0 ? (
                          latestAnalysis.extracted_dates.map((dateItem) => (
                            <div key={`${dateItem.label}-${dateItem.source}`} className="rounded-lg border px-3 py-2 text-sm">
                              <div className="font-medium">{dateItem.label}</div>
                              <div className="text-muted-foreground">
                                {dateItem.value ? formatDateTime(dateItem.value) : 'No value'} • {dateItem.source}
                              </div>
                            </div>
                          ))
                        ) : (
                          <p className="text-sm text-muted-foreground">No dates extracted.</p>
                        )}
                      </div>
                    </div>
                  </SectionCard>
                </div>

                <div className="grid grid-cols-1 gap-4 xl:grid-cols-[1fr_1fr]">
                  <SectionCard title="Key Findings" description="Top contract findings with recommendations.">
                    <div className="space-y-3">
                      {latestAnalysis.key_findings.length > 0 ? (
                        latestAnalysis.key_findings.map((finding) => (
                          <div key={`${finding.title}-${finding.clause_reference ?? 'none'}`} className="rounded-lg border px-4 py-3">
                            <div className="flex items-start justify-between gap-3">
                              <div>
                                <p className="font-medium">{finding.title}</p>
                                <p className="mt-1 text-sm text-muted-foreground">{finding.description}</p>
                                <p className="mt-2 text-sm">
                                  Recommendation: <span className="text-muted-foreground">{finding.recommendation}</span>
                                </p>
                              </div>
                              <SeverityIndicator severity={normalizeRiskSeverity(finding.severity)} size="sm" />
                            </div>
                          </div>
                        ))
                      ) : (
                        <p className="text-sm text-muted-foreground">No key findings were returned.</p>
                      )}
                    </div>
                  </SectionCard>

                  <SectionCard title="Missing Clauses & Flags" description="Gaps and compliance flags detected in the latest run.">
                    <div className="space-y-4">
                      <div className="space-y-2">
                        <p className="text-sm font-medium">Missing clauses</p>
                        {latestAnalysis.missing_clauses.length > 0 ? (
                          <div className="flex flex-wrap gap-2">
                            {latestAnalysis.missing_clauses.map((clauseType) => (
                              <Badge key={clauseType} variant="warning">
                                {titleCase(clauseType)}
                              </Badge>
                            ))}
                          </div>
                        ) : (
                          <p className="text-sm text-muted-foreground">No missing clauses detected.</p>
                        )}
                      </div>

                      <div className="space-y-2">
                        <p className="text-sm font-medium">Compliance flags</p>
                        {latestAnalysis.compliance_flags.length > 0 ? (
                          latestAnalysis.compliance_flags.map((flag) => (
                            <div key={`${flag.code}-${flag.title}`} className="rounded-lg border px-4 py-3">
                              <div className="flex items-start justify-between gap-3">
                                <div>
                                  <p className="font-medium">{flag.title}</p>
                                  <p className="text-sm text-muted-foreground">{flag.description}</p>
                                  <p className="mt-1 text-xs text-muted-foreground">{flag.code}</p>
                                </div>
                                <SeverityIndicator severity={normalizeRiskSeverity(flag.severity)} size="sm" />
                              </div>
                            </div>
                          ))
                        ) : (
                          <p className="text-sm text-muted-foreground">No compliance flags were raised.</p>
                        )}
                      </div>
                    </div>
                  </SectionCard>
                </div>
              </>
            ) : (
              <SectionCard title="Analysis" description="The current contract does not have a stored analysis result yet.">
                <EmptyState
                  icon={FileSearch}
                  title="No analysis available"
                  description="Run a contract analysis to populate clause extraction, risk scoring, and compliance signals."
                  action={{
                    label: analyzeMutation.isPending ? 'Analyzing…' : 'Analyze Contract',
                    onClick: () => void analyzeMutation.mutate(),
                  }}
                />
              </SectionCard>
            )}

            <SectionCard title="Clauses" description="Clause-by-clause review state, summaries, and review workflow.">
              <div className="space-y-3">
                {clauses.length > 0 ? (
                  clauses.map((clause) => (
                    <div key={clause.id} className="rounded-lg border px-4 py-3">
                      <div className="flex flex-wrap items-start justify-between gap-3">
                        <div className="space-y-2">
                          <div className="flex flex-wrap items-center gap-2">
                            <p className="font-medium">{clause.title}</p>
                            <Badge variant="outline">{titleCase(clause.clause_type)}</Badge>
                            <StatusBadge status={clause.review_status} config={clauseReviewStatusConfig} size="sm" />
                          </div>
                          <p className="text-sm text-muted-foreground">
                            {clause.analysis_summary || clause.content.slice(0, 220) || 'No analysis summary available.'}
                          </p>
                          <div className="flex flex-wrap gap-4 text-xs text-muted-foreground">
                            <span>Risk score: {formatNumber(clause.risk_score)}</span>
                            <span>Confidence: {Math.round(clause.extraction_confidence * 100)}%</span>
                            <span>{clause.section_reference || 'No section reference'}</span>
                          </div>
                          {clause.recommendations.length > 0 ? (
                            <div className="flex flex-wrap gap-2">
                              {clause.recommendations.map((recommendation) => (
                                <Badge key={recommendation} variant="secondary">
                                  {recommendation}
                                </Badge>
                              ))}
                            </div>
                          ) : null}
                        </div>
                        <div className="flex items-center gap-2">
                          <SeverityIndicator severity={normalizeRiskSeverity(clause.risk_level)} size="sm" />
                          {canWrite ? (
                            <Button variant="outline" size="sm" onClick={() => setClauseReviewTarget(clause)}>
                              Review Clause
                            </Button>
                          ) : null}
                        </div>
                      </div>
                    </div>
                  ))
                ) : (
                  <p className="text-sm text-muted-foreground">No clauses are available for this contract yet.</p>
                )}
              </div>
            </SectionCard>
          </TabsContent>

          <TabsContent value="versions" className="space-y-4">
            <SectionCard
              title="Version History"
              description="Uploaded contract versions with file hashes and change summaries."
              actions={
                canWrite ? (
                  <Button size="sm" onClick={() => setUploadOpen(true)}>
                    <FileUp className="mr-1.5 h-3.5 w-3.5" />
                    Upload Version
                  </Button>
                ) : undefined
              }
            >
              {versionsQuery.isLoading ? (
                <LoadingSkeleton variant="list-item" count={3} />
              ) : versionsQuery.isError ? (
                <ErrorState
                  message="Failed to load contract versions."
                  onRetry={() => void versionsQuery.refetch()}
                />
              ) : versions.length > 0 ? (
                <div className="space-y-3">
                  {versions.map((version) => (
                    <div key={version.id} className="rounded-lg border px-4 py-3">
                      <div className="flex flex-wrap items-start justify-between gap-3">
                        <div className="space-y-2">
                          <div className="flex flex-wrap items-center gap-2">
                            <p className="font-medium">Version {version.version}</p>
                            <Badge variant="outline">{version.file_name}</Badge>
                          </div>
                          <div className="text-sm text-muted-foreground">
                            {version.change_summary || 'No change summary recorded.'}
                          </div>
                          <div className="flex flex-wrap gap-4 text-xs text-muted-foreground">
                            <span>{formatBytes(version.file_size_bytes)}</span>
                            <span>{formatDateTime(version.uploaded_at)}</span>
                            <span>SHA-256 {version.content_hash.slice(0, 12)}…</span>
                          </div>
                        </div>
                        <Button variant="outline" size="sm" onClick={() => void downloadVersion(version)}>
                          Download
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">No versions have been uploaded yet.</p>
              )}
            </SectionCard>
          </TabsContent>

          <TabsContent value="workflow" className="space-y-4">
            {contract.workflow_instance_id ? (
              <SectionCard title="Workflow Linkage" description="Contract review state is persisted in the workflow engine.">
                <div className="space-y-3">
                  <MetadataRow
                    label="Workflow instance"
                    value={
                      <Link
                        href={`/workflows/instances/${contract.workflow_instance_id}`}
                        className="inline-flex items-center gap-1 text-primary hover:underline"
                      >
                        {contract.workflow_instance_id}
                        <ArrowUpRight className="h-3.5 w-3.5" />
                      </Link>
                    }
                  />
                  <MetadataRow label="Contract status" value={titleCase(contract.status)} />
                  <MetadataRow label="Current version" value={`v${contract.current_version}`} />
                  <MetadataRow
                    label="Started"
                    value={contract.status_changed_at ? formatDateTime(contract.status_changed_at) : 'Not available'}
                  />
                </div>
              </SectionCard>
            ) : (
              <SectionCard title="Workflow Linkage" description="This contract has not entered a review workflow yet.">
                <EmptyState
                  icon={GitBranch}
                  title="No workflow linked"
                  description="Start a review workflow to create a tenant-scoped human task in the workflow engine."
                  action={
                    canWrite
                      ? {
                          label: 'Start Review Workflow',
                          onClick: () => setReviewOpen(true),
                        }
                      : undefined
                  }
                />
              </SectionCard>
            )}
          </TabsContent>
        </Tabs>

        <ContractFormDialog
          contract={contract}
          open={editOpen}
          onOpenChange={setEditOpen}
          onSaved={() => {
            void refreshContract();
          }}
        />

        <StatusDialog
          currentStatus={contract.status}
          loading={statusMutation.isPending}
          onOpenChange={setStatusOpen}
          onSubmit={(nextStatus) => statusMutation.mutate(nextStatus)}
          open={statusOpen}
          options={allowedStatuses}
        />

        <ContractVersionUploadDialog
          contract={contract}
          loading={versionsQuery.isFetching}
          onOpenChange={setUploadOpen}
          onSaved={() => {
            void refreshContract();
          }}
          open={uploadOpen}
        />

        <ReviewDialog
          loading={reviewMutation.isPending}
          onOpenChange={setReviewOpen}
          onSubmit={(payload) => reviewMutation.mutate(payload)}
          open={reviewOpen}
          users={users}
          usersLoading={usersQuery.isLoading}
        />

        <RenewDialog
          contract={contract}
          loading={renewMutation.isPending}
          onOpenChange={setRenewOpen}
          onSubmit={(payload) => renewMutation.mutate(payload)}
          open={renewOpen}
        />

        <ClauseReviewDialog
          clause={clauseReviewTarget}
          loading={clauseReviewMutation.isPending}
          onOpenChange={(open) => {
            if (!open) {
              setClauseReviewTarget(null);
            }
          }}
          onSubmit={(draft) => {
            if (!clauseReviewTarget) {
              return;
            }
            clauseReviewMutation.mutate({
              clauseId: clauseReviewTarget.id,
              notes: draft.notes,
              status: draft.status,
            });
          }}
          open={Boolean(clauseReviewTarget)}
        />

        <ConfirmDialog
          open={deleteOpen}
          onOpenChange={setDeleteOpen}
          title="Delete Contract"
          description={`Delete "${contract.title}"? This removes the contract from the active portfolio.`}
          confirmLabel="Delete Contract"
          variant="destructive"
          loading={deleteMutation.isPending}
          onConfirm={async () => {
            await deleteMutation.mutateAsync();
          }}
        />
      </div>
    </PermissionRedirect>
  );
}

function MetricCard({
  helper,
  label,
  value,
}: {
  helper?: string;
  label: string;
  value: React.ReactNode;
}) {
  return (
    <div className="rounded-lg border bg-card p-4">
      <p className="text-sm text-muted-foreground">{label}</p>
      <div className="mt-2 text-lg font-semibold">{value}</div>
      {helper ? <p className="mt-2 text-xs text-muted-foreground">{helper}</p> : null}
    </div>
  );
}

function MetadataRow({
  label,
  value,
}: {
  label: string;
  value: React.ReactNode;
}) {
  return (
    <div className="grid grid-cols-1 gap-1 sm:grid-cols-[160px_1fr]">
      <span className="text-sm text-muted-foreground">{label}</span>
      <div className="text-sm">{value}</div>
    </div>
  );
}

function StatusDialog({
  currentStatus,
  loading,
  onOpenChange,
  onSubmit,
  open,
  options,
}: {
  currentStatus: LexContractStatus;
  loading: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (status: LexContractStatus) => void;
  open: boolean;
  options: LexContractStatus[];
}) {
  const [status, setStatus] = useState<LexContractStatus | ''>('');

  useEffect(() => {
    if (!open) {
      setStatus('');
      return;
    }
    setStatus(options[0] ?? '');
  }, [open, options]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Change Status</DialogTitle>
          <DialogDescription>
            Move the contract from {titleCase(currentStatus)} to a valid next state.
          </DialogDescription>
        </DialogHeader>

        {options.length > 0 ? (
          <div className="space-y-3">
            <Label htmlFor="next-status">Next status</Label>
            <Select value={status} onValueChange={(value) => setStatus(value as LexContractStatus)}>
              <SelectTrigger id="next-status">
                <SelectValue placeholder="Select status" />
              </SelectTrigger>
              <SelectContent>
                {options.map((option) => (
                  <SelectItem key={option} value={option}>
                    {titleCase(option)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">
            This contract has no further status transitions from its current state.
          </p>
        )}

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            type="button"
            disabled={!status || loading || options.length === 0}
            onClick={() => status && onSubmit(status)}
          >
            Update Status
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ReviewDialog({
  loading,
  onOpenChange,
  onSubmit,
  open,
  users,
  usersLoading,
}: {
  loading: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (payload: Record<string, unknown>) => void;
  open: boolean;
  users: UserDirectoryEntry[];
  usersLoading: boolean;
}) {
  const [draft, setDraft] = useState<ReviewDraft>({
    approverRole: 'legal',
    approverUserId: '',
    description: '',
    slaHours: '48',
  });

  useEffect(() => {
    if (!open) {
      setDraft({
        approverRole: 'legal',
        approverUserId: '',
        description: '',
        slaHours: '48',
      });
    }
  }, [open]);

  const isValid = draft.description.trim().length >= 5 && (draft.approverRole.trim() || draft.approverUserId);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Start Review Workflow</DialogTitle>
          <DialogDescription>
            Create a workflow-backed legal review task for this contract.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="review-user">Specific approver</Label>
            <Select
              value={draft.approverUserId || 'none'}
              onValueChange={(value) =>
                setDraft((current) => ({
                  ...current,
                  approverUserId: value === 'none' ? '' : value,
                }))
              }
            >
              <SelectTrigger id="review-user">
                <SelectValue placeholder={usersLoading ? 'Loading users…' : 'Select approver'} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">Assign by role</SelectItem>
                {users.map((user) => (
                  <SelectItem key={user.id} value={user.id}>
                    {userDisplayName(user)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="review-role">Approver role</Label>
            <Input
              id="review-role"
              value={draft.approverRole}
              onChange={(event) =>
                setDraft((current) => ({ ...current, approverRole: event.target.value }))
              }
              placeholder="legal"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="review-sla">SLA hours</Label>
            <Input
              id="review-sla"
              type="number"
              min={1}
              value={draft.slaHours}
              onChange={(event) =>
                setDraft((current) => ({ ...current, slaHours: event.target.value }))
              }
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="review-description">Task description</Label>
            <Textarea
              id="review-description"
              value={draft.description}
              onChange={(event) =>
                setDraft((current) => ({ ...current, description: event.target.value }))
              }
              placeholder="Review key obligations, clause risks, and approval readiness."
              rows={4}
            />
          </div>
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            type="button"
            disabled={!isValid || loading}
            onClick={() =>
              onSubmit({
                approver_role: draft.approverRole.trim() || undefined,
                approver_user_id: draft.approverUserId || undefined,
                description: draft.description.trim(),
                sla_hours: Number(draft.slaHours || '48'),
              })
            }
          >
            Start Review
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function RenewDialog({
  contract,
  loading,
  onOpenChange,
  onSubmit,
  open,
}: {
  contract: LexContractRecord;
  loading: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (payload: Record<string, unknown>) => void;
  open: boolean;
}) {
  const [draft, setDraft] = useState<RenewDraft>({
    changeSummary: '',
    newEffectiveDate: '',
    newExpiryDate: toDateInputValue(contract.expiry_date),
    newValue: contract.total_value != null ? String(contract.total_value) : '',
  });

  useEffect(() => {
    if (open) {
      setDraft({
        changeSummary: '',
        newEffectiveDate: '',
        newExpiryDate: toDateInputValue(contract.expiry_date),
        newValue: contract.total_value != null ? String(contract.total_value) : '',
      });
    }
  }, [contract.expiry_date, contract.total_value, open]);

  const isValid = draft.changeSummary.trim().length >= 3 && draft.newExpiryDate;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Renew Contract</DialogTitle>
          <DialogDescription>
            Create a renewal record with updated dates and commercial terms.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="renew-effective">New effective date</Label>
            <Input
              id="renew-effective"
              type="date"
              value={draft.newEffectiveDate}
              onChange={(event) =>
                setDraft((current) => ({ ...current, newEffectiveDate: event.target.value }))
              }
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="renew-expiry">New expiry date</Label>
            <Input
              id="renew-expiry"
              type="date"
              value={draft.newExpiryDate}
              onChange={(event) =>
                setDraft((current) => ({ ...current, newExpiryDate: event.target.value }))
              }
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="renew-value">New value</Label>
            <Input
              id="renew-value"
              type="number"
              min={0}
              step="0.01"
              value={draft.newValue}
              onChange={(event) =>
                setDraft((current) => ({ ...current, newValue: event.target.value }))
              }
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="renew-summary">Change summary</Label>
            <Textarea
              id="renew-summary"
              value={draft.changeSummary}
              onChange={(event) =>
                setDraft((current) => ({ ...current, changeSummary: event.target.value }))
              }
              placeholder="Annual renewal with updated commercial rates."
              rows={3}
            />
          </div>
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            type="button"
            disabled={!isValid || loading}
            onClick={() =>
              onSubmit({
                new_effective_date: toOptionalDateTime(draft.newEffectiveDate),
                new_expiry_date: requiredDateTime(draft.newExpiryDate),
                new_value: draft.newValue === '' ? null : Number(draft.newValue),
                change_summary: draft.changeSummary.trim(),
              })
            }
          >
            Renew Contract
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ContractVersionUploadDialog({
  contract,
  loading,
  onOpenChange,
  onSaved,
  open,
}: {
  contract: LexContractRecord;
  loading: boolean;
  onOpenChange: (open: boolean) => void;
  onSaved: () => void;
  open: boolean;
}) {
  const [file, setFile] = useState<File | null>(null);
  const [extractedText, setExtractedText] = useState('');
  const [changeSummary, setChangeSummary] = useState('');
  const [uploadProgress, setUploadProgress] = useState(0);

  const uploadMutation = useMutation({
    mutationFn: async () => {
      if (!file) {
        throw new Error('Select a file before uploading a new version.');
      }
      const uploaded = await enterpriseApi.files.upload(
        file,
        {
          suite: 'lex',
          entity_type: 'contract',
          entity_id: contract.id,
          tags: Array.from(new Set(['contract', contract.type, ...contract.tags])).join(','),
          lifecycle_policy: 'standard',
        },
        setUploadProgress,
      );
      return enterpriseApi.lex.uploadContractDocument(contract.id, {
        file_id: uploaded.id,
        file_name: uploaded.original_name,
        file_size_bytes: uploaded.size_bytes,
        content_hash: uploaded.checksum_sha256,
        extracted_text: extractedText.trim(),
        change_summary: changeSummary.trim(),
      });
    },
    onSuccess: () => {
      showSuccess('Version uploaded.', 'The contract version history has been updated.');
      setFile(null);
      setExtractedText('');
      setChangeSummary('');
      setUploadProgress(0);
      onOpenChange(false);
      onSaved();
    },
    onError: showApiError,
  });

  useEffect(() => {
    if (!open) {
      setFile(null);
      setExtractedText('');
      setChangeSummary('');
      setUploadProgress(0);
    }
  }, [open]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Upload New Version</DialogTitle>
          <DialogDescription>
            Attach a new contract file and optional extracted text for analysis.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="contract-version-file">Contract file</Label>
            <Input
              id="contract-version-file"
              type="file"
              accept=".pdf,.doc,.docx,.txt"
              onChange={(event) => setFile(event.target.files?.[0] ?? null)}
            />
            {file ? <p className="text-xs text-muted-foreground">Selected: {file.name}</p> : null}
          </div>

          <div className="space-y-2">
            <Label htmlFor="contract-version-summary">Change summary</Label>
            <Input
              id="contract-version-summary"
              value={changeSummary}
              onChange={(event) => setChangeSummary(event.target.value)}
              placeholder="Updated commercial schedule and renewal appendix."
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="contract-version-text">Extracted text</Label>
            <Textarea
              id="contract-version-text"
              value={extractedText}
              onChange={(event) => setExtractedText(event.target.value)}
              placeholder="Paste contract text if you want immediate deterministic analysis."
              rows={5}
            />
          </div>

          {uploadMutation.isPending ? (
            <p className="text-xs text-muted-foreground">
              Upload progress: {Math.round(uploadProgress)}%
            </p>
          ) : null}
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            type="button"
            disabled={!file || uploadMutation.isPending || loading}
            onClick={() => uploadMutation.mutate()}
          >
            Upload Version
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ClauseReviewDialog({
  clause,
  loading,
  onOpenChange,
  onSubmit,
  open,
}: {
  clause: LexClause | null;
  loading: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (draft: ClauseReviewDraft) => void;
  open: boolean;
}) {
  const [draft, setDraft] = useState<ClauseReviewDraft>({
    notes: '',
    status: 'reviewed',
  });

  useEffect(() => {
    if (clause && open) {
      setDraft({
        notes: clause.review_notes ?? '',
        status: clause.review_status,
      });
    }
  }, [clause, open]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Review Clause</DialogTitle>
          <DialogDescription>
            Persist a review decision for {clause?.title ?? 'this clause'}.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="clause-status">Review status</Label>
            <Select
              value={draft.status}
              onValueChange={(value) =>
                setDraft((current) => ({
                  ...current,
                  status: value as LexClause['review_status'],
                }))
              }
            >
              <SelectTrigger id="clause-status">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {['pending', 'reviewed', 'flagged', 'accepted', 'rejected'].map((status) => (
                  <SelectItem key={status} value={status}>
                    {titleCase(status)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="clause-notes">Review notes</Label>
            <Textarea
              id="clause-notes"
              value={draft.notes}
              onChange={(event) =>
                setDraft((current) => ({ ...current, notes: event.target.value }))
              }
              placeholder="Document the legal reasoning behind the clause decision."
              rows={4}
            />
          </div>
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            type="button"
            disabled={draft.notes.trim().length < 3 || loading}
            onClick={() => onSubmit({ ...draft, notes: draft.notes.trim() })}
          >
            Save Review
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function normalizeRiskSeverity(
  value: string | null | undefined,
): 'critical' | 'high' | 'medium' | 'low' | 'info' {
  switch (value) {
    case 'critical':
    case 'high':
    case 'medium':
    case 'low':
      return value;
    default:
      return 'info';
  }
}

function formatOptionalDate(value?: string | null): string {
  return value ? formatDateTime(value) : 'Not set';
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

function requiredDateTime(value: string): string {
  return new Date(`${value}T00:00:00Z`).toISOString();
}

async function downloadVersion(version: LexContractVersion): Promise<void> {
  const blob = await enterpriseApi.files.download(version.file_id);
  downloadBlob(blob, version.file_name);
}
