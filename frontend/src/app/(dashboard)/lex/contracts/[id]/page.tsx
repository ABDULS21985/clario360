'use client';

import { useState } from 'react';
import { useParams } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { FileText, ShieldCheck } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { ErrorState } from '@/components/common/error-state';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { PageHeader } from '@/components/common/page-header';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { StatusBadge } from '@/components/shared/status-badge';
import { SectionCard } from '@/components/suites/section-card';
import { apiPost } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { fetchSuiteData, type SuiteEnvelope } from '@/lib/suite-api';
import { contractStatusConfig } from '@/lib/status-configs';
import { showApiError, showSuccess } from '@/lib/toast';
import { summarizeNamedRecords } from '@/lib/suite-utils';
import { formatDateTime } from '@/lib/utils';
import type { ComplianceCheckResult, LexContract } from '@/types/suites';

export default function LexContractDetailPage() {
  const params = useParams<{ id: string }>();
  const id = params?.id ?? '';
  const [checkResults, setCheckResults] = useState<ComplianceCheckResult[] | null>(null);
  const [checking, setChecking] = useState(false);

  const contractQuery = useQuery({
    queryKey: ['lex-contract', id],
    queryFn: () => fetchSuiteData<LexContract>(`${API_ENDPOINTS.LEX_CONTRACTS}/${id}`),
    enabled: Boolean(id),
  });

  const runComplianceCheck = async () => {
    if (!id) {
      return;
    }
    try {
      setChecking(true);
      const response = await apiPost<SuiteEnvelope<ComplianceCheckResult[]>>(API_ENDPOINTS.LEX_COMPLIANCE_CHECK, {
        entity_type: 'contract',
        entity_id: id,
      });
      setCheckResults(response.data);
      showSuccess('Compliance check completed.', `${response.data.length} rule evaluations returned.`);
    } catch (error) {
      showApiError(error);
    } finally {
      setChecking(false);
    }
  };

  if (contractQuery.isLoading) {
    return (
      <PermissionRedirect permission="lex:read">
        <div className="space-y-6">
          <PageHeader title="Contract Details" description={`Contract ID: ${id}`} />
          <LoadingSkeleton variant="card" count={3} />
        </div>
      </PermissionRedirect>
    );
  }

  if (contractQuery.error || !contractQuery.data) {
    return (
      <PermissionRedirect permission="lex:read">
        <ErrorState message="Failed to load contract details." onRetry={() => void contractQuery.refetch()} />
      </PermissionRedirect>
    );
  }

  const contract = contractQuery.data;

  return (
    <PermissionRedirect permission="lex:read">
      <div className="space-y-6">
        <PageHeader
          title={contract.title}
          description={`Contract type: ${contract.type.replace(/_/g, ' ')}`}
          actions={
            <Button size="sm" onClick={() => void runComplianceCheck()} disabled={checking}>
              <ShieldCheck className="mr-1.5 h-3.5 w-3.5" />
              {checking ? 'Checking…' : 'Run Compliance Check'}
            </Button>
          }
        />

        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <div className="rounded-lg border bg-card p-4">
            <p className="text-sm text-muted-foreground">Status</p>
            <div className="mt-2">
              <StatusBadge status={contract.status} config={contractStatusConfig} />
            </div>
          </div>
          <div className="rounded-lg border bg-card p-4">
            <p className="text-sm text-muted-foreground">Parties</p>
            <p className="mt-2 text-lg font-semibold">{summarizeNamedRecords(contract.parties, 3)}</p>
          </div>
          <div className="rounded-lg border bg-card p-4">
            <p className="text-sm text-muted-foreground">Value</p>
            <p className="mt-2 text-lg font-semibold">
              {contract.value != null ? `${contract.currency} ${contract.value.toLocaleString()}` : 'Undisclosed'}
            </p>
          </div>
          <div className="rounded-lg border bg-card p-4">
            <p className="text-sm text-muted-foreground">Expiry</p>
            <p className="mt-2 text-lg font-semibold">{contract.expiry_date ? formatDateTime(contract.expiry_date) : 'No expiry'}</p>
          </div>
        </div>

        <div className="grid gap-4 xl:grid-cols-[1.05fr_0.95fr]">
          <SectionCard title="Contract Metadata" description="Dates, file references, and structured metadata.">
            <div className="space-y-3">
              <div className="rounded-lg border px-4 py-3">
                <p className="text-sm text-muted-foreground">Effective Date</p>
                <p className="mt-1 font-medium">{contract.effective_date ? formatDateTime(contract.effective_date) : 'Not set'}</p>
              </div>
              <div className="rounded-lg border px-4 py-3">
                <p className="text-sm text-muted-foreground">File URL</p>
                {contract.file_url ? (
                  <a href={contract.file_url} target="_blank" rel="noreferrer" className="mt-1 inline-block text-sm text-primary hover:underline">
                    Open contract file
                  </a>
                ) : (
                  <p className="mt-1 text-sm text-muted-foreground">No file reference attached.</p>
                )}
              </div>
              <div className="rounded-lg border px-4 py-3">
                <p className="text-sm text-muted-foreground">Metadata</p>
                <pre className="mt-2 overflow-x-auto whitespace-pre-wrap text-xs text-muted-foreground">
                  {JSON.stringify(contract.metadata, null, 2)}
                </pre>
              </div>
            </div>
          </SectionCard>

          <SectionCard title="Counterparties" description="Structured party records attached to the contract.">
            <div className="space-y-3">
              {Array.isArray(contract.parties) && contract.parties.length > 0 ? (
                contract.parties.map((party, index) => (
                  <div key={`${index}-${JSON.stringify(party)}`} className="rounded-lg border px-4 py-3 text-sm">
                    {summarizeNamedRecords([party], 1)}
                  </div>
                ))
              ) : (
                <p className="text-sm text-muted-foreground">No structured parties are attached to this contract.</p>
              )}
            </div>
          </SectionCard>
        </div>

        <SectionCard title="Compliance Check Results" description="On-demand evaluations against current regulatory rules.">
          {checkResults === null ? (
            <p className="text-sm text-muted-foreground">Run a compliance check to see live rule evaluation results for this contract.</p>
          ) : checkResults.length === 0 ? (
            <p className="text-sm text-muted-foreground">No enabled rules applied to this contract.</p>
          ) : (
            <div className="space-y-3">
              {checkResults.map((result) => (
                <div key={result.rule_id} className="rounded-lg border px-4 py-3">
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <div className="flex items-center gap-2">
                        <FileText className="h-4 w-4 text-muted-foreground" />
                        <p className="font-medium">{result.rule_name}</p>
                      </div>
                      <p className="mt-2 text-sm text-muted-foreground">{result.message}</p>
                    </div>
                    <div className="flex items-center gap-2">
                      <SeverityIndicator severity={normalizeSeverity(result.severity)} size="sm" />
                      <Badge variant={result.status === 'compliant' ? 'success' : 'warning'}>
                        {result.status.replace(/_/g, ' ')}
                      </Badge>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </SectionCard>
      </div>
    </PermissionRedirect>
  );
}

function normalizeSeverity(value: string): 'critical' | 'high' | 'medium' | 'low' | 'info' {
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
