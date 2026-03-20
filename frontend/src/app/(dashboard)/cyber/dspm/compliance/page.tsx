'use client';

import { useState, useMemo } from 'react';
import {
  AlertTriangle,
  CheckCircle2,
  ChevronDown,
  ChevronUp,
  Globe,
  Shield,
  ShieldAlert,
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { PermissionRedirect } from '@/components/common/permission-redirect';
import { useRealtimeData } from '@/hooks/use-realtime-data';
import { apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { DSPMPolicyViolation, DSPMDataPolicy } from '@/types/cyber';

const SEVERITY_COLORS: Record<string, string> = {
  critical: 'bg-red-100 text-red-700',
  high: 'bg-orange-100 text-orange-700',
  medium: 'bg-amber-100 text-amber-800',
  low: 'bg-blue-100 text-blue-700',
  info: 'bg-gray-100 text-gray-600',
};

const FRAMEWORK_METADATA: Record<string, { label: string; color: string; icon: typeof Shield }> = {
  GDPR: { label: 'GDPR', color: 'border-blue-300 bg-blue-50', icon: Globe },
  HIPAA: { label: 'HIPAA', color: 'border-green-300 bg-green-50', icon: Shield },
  SOC2: { label: 'SOC 2', color: 'border-purple-300 bg-purple-50', icon: Shield },
  'PCI-DSS': { label: 'PCI-DSS', color: 'border-amber-300 bg-amber-50', icon: ShieldAlert },
  'SAUDI PDPL': { label: 'Saudi PDPL', color: 'border-teal-300 bg-teal-50', icon: Globe },
  'PCI_DSS': { label: 'PCI-DSS', color: 'border-amber-300 bg-amber-50', icon: ShieldAlert },
  PDPL: { label: 'Saudi PDPL', color: 'border-teal-300 bg-teal-50', icon: Globe },
};

interface FrameworkData {
  name: string;
  violations: DSPMPolicyViolation[];
  severityBreakdown: Record<string, number>;
}

export default function CompliancePosturePage() {
  const [selectedFramework, setSelectedFramework] = useState<string | null>(null);

  const {
    data: violationsEnvelope,
    isLoading: violationsLoading,
    error: violationsError,
    mutate: refetchViolations,
  } = useRealtimeData<{ data: DSPMPolicyViolation[] }>(API_ENDPOINTS.CYBER_DSPM_POLICY_VIOLATIONS, {
    pollInterval: 120000,
  });

  const {
    data: policiesEnvelope,
    isLoading: policiesLoading,
  } = useRealtimeData<{ data: DSPMDataPolicy[] }>(API_ENDPOINTS.CYBER_DSPM_DATA_POLICIES, {
    pollInterval: 120000,
  });

  const violations = violationsEnvelope?.data ?? [];
  const policies = policiesEnvelope?.data ?? [];
  const isLoading = violationsLoading || policiesLoading;

  const { frameworks, totalViolations, frameworkCount, criticalViolations } = useMemo(() => {
    const frameworkMap: Record<string, DSPMPolicyViolation[]> = {};

    for (const v of violations) {
      const complianceFrameworks = v.compliance_frameworks ?? [];
      if (complianceFrameworks.length === 0) {
        const key = 'Uncategorized';
        if (!frameworkMap[key]) frameworkMap[key] = [];
        frameworkMap[key].push(v);
      } else {
        for (const fw of complianceFrameworks) {
          const key = fw.toUpperCase();
          if (!frameworkMap[key]) frameworkMap[key] = [];
          frameworkMap[key].push(v);
        }
      }
    }

    // Also group by frameworks referenced in policies
    for (const p of policies) {
      const policyFrameworks = p.compliance_frameworks ?? [];
      for (const fw of policyFrameworks) {
        const key = fw.toUpperCase();
        if (!frameworkMap[key]) frameworkMap[key] = [];
      }
    }

    const fws: FrameworkData[] = Object.entries(frameworkMap).map(([name, fvs]) => {
      const severityBreakdown: Record<string, number> = {};
      for (const v of fvs) {
        severityBreakdown[v.severity] = (severityBreakdown[v.severity] ?? 0) + 1;
      }
      return { name, violations: fvs, severityBreakdown };
    });

    fws.sort((a, b) => b.violations.length - a.violations.length);

    const critical = violations.filter((v) => v.severity === 'critical').length;

    return {
      frameworks: fws,
      totalViolations: violations.length,
      frameworkCount: fws.length,
      criticalViolations: critical,
    };
  }, [violations, policies]);

  const selectedFrameworkData = useMemo(() => {
    if (!selectedFramework) return null;
    return frameworks.find((f) => f.name === selectedFramework) ?? null;
  }, [selectedFramework, frameworks]);

  function toggleFramework(name: string) {
    setSelectedFramework((prev) => (prev === name ? null : name));
  }

  return (
    <PermissionRedirect permission="cyber:read">
      <div className="space-y-6">
        <PageHeader
          title="Compliance Posture"
          description="Monitor data security compliance across regulatory frameworks and industry standards"
        />

        {isLoading ? (
          <LoadingSkeleton variant="card" count={3} />
        ) : violationsError ? (
          <ErrorState message="Failed to load compliance data" onRetry={() => void refetchViolations()} />
        ) : (
          <>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <Card>
                <CardContent className="flex items-center gap-4 p-5">
                  <AlertTriangle className="h-5 w-5 text-red-600" />
                  <div>
                    <p className="text-xs text-muted-foreground">Total Violations</p>
                    <p className="text-2xl font-bold tabular-nums">{totalViolations}</p>
                  </div>
                </CardContent>
              </Card>
              <Card>
                <CardContent className="flex items-center gap-4 p-5">
                  <Shield className="h-5 w-5 text-blue-600" />
                  <div>
                    <p className="text-xs text-muted-foreground">Frameworks Covered</p>
                    <p className="text-2xl font-bold tabular-nums">{frameworkCount}</p>
                  </div>
                </CardContent>
              </Card>
              <Card>
                <CardContent className="flex items-center gap-4 p-5">
                  <ShieldAlert className="h-5 w-5 text-red-600" />
                  <div>
                    <p className="text-xs text-muted-foreground">Critical Violations</p>
                    <p className="text-2xl font-bold tabular-nums">{criticalViolations}</p>
                  </div>
                </CardContent>
              </Card>
            </div>

            {frameworks.length === 0 ? (
              <Card>
                <CardContent className="flex flex-col items-center justify-center py-12 text-center">
                  <CheckCircle2 className="mb-3 h-8 w-8 text-green-500" />
                  <p className="text-sm font-medium">No Compliance Violations</p>
                  <p className="text-xs text-muted-foreground">All data assets are compliant across all frameworks.</p>
                </CardContent>
              </Card>
            ) : (
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
                {frameworks.map((fw) => {
                  const meta = FRAMEWORK_METADATA[fw.name] ?? { label: fw.name, color: 'border-gray-300 bg-gray-50', icon: Shield };
                  const FrameworkIcon = meta.icon;
                  const isSelected = selectedFramework === fw.name;
                  const critCount = fw.severityBreakdown['critical'] ?? 0;
                  const highCount = fw.severityBreakdown['high'] ?? 0;
                  const medCount = fw.severityBreakdown['medium'] ?? 0;
                  const lowCount = fw.severityBreakdown['low'] ?? 0;

                  return (
                    <Card
                      key={fw.name}
                      className={`cursor-pointer transition-all hover:shadow-md ${isSelected ? 'ring-2 ring-primary' : ''} ${meta.color}`}
                      onClick={() => toggleFramework(fw.name)}
                    >
                      <CardHeader className="pb-2">
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-2">
                            <FrameworkIcon className="h-5 w-5 text-muted-foreground" />
                            <CardTitle className="text-sm">{meta.label}</CardTitle>
                          </div>
                          {isSelected ? (
                            <ChevronUp className="h-4 w-4 text-muted-foreground" />
                          ) : (
                            <ChevronDown className="h-4 w-4 text-muted-foreground" />
                          )}
                        </div>
                      </CardHeader>
                      <CardContent className="space-y-3">
                        <div className="flex items-center justify-between">
                          <span className="text-xs text-muted-foreground">Violations</span>
                          <span className={`text-lg font-bold tabular-nums ${fw.violations.length > 0 ? 'text-red-600' : 'text-green-600'}`}>
                            {fw.violations.length}
                          </span>
                        </div>
                        <div className="flex flex-wrap gap-2">
                          {critCount > 0 && (
                            <span className="inline-flex items-center gap-1 rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-700">
                              {critCount} Critical
                            </span>
                          )}
                          {highCount > 0 && (
                            <span className="inline-flex items-center gap-1 rounded-full bg-orange-100 px-2 py-0.5 text-xs font-medium text-orange-700">
                              {highCount} High
                            </span>
                          )}
                          {medCount > 0 && (
                            <span className="inline-flex items-center gap-1 rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-800">
                              {medCount} Medium
                            </span>
                          )}
                          {lowCount > 0 && (
                            <span className="inline-flex items-center gap-1 rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-700">
                              {lowCount} Low
                            </span>
                          )}
                          {fw.violations.length === 0 && (
                            <span className="inline-flex items-center gap-1 text-xs text-green-600">
                              <CheckCircle2 className="h-3 w-3" />
                              Compliant
                            </span>
                          )}
                        </div>
                        {fw.violations.length > 0 && (
                          <div className="space-y-1">
                            <p className="text-xs font-medium text-muted-foreground">Top Violations</p>
                            {fw.violations.slice(0, 3).map((v, idx) => (
                              <div key={`${v.policy_id}-${v.asset_id}-${idx}`} className="flex items-center justify-between text-xs">
                                <span className="truncate pr-2">{v.policy_name}</span>
                                <span className={`inline-flex shrink-0 rounded-full px-1.5 py-0.5 font-medium capitalize ${SEVERITY_COLORS[v.severity] ?? 'bg-muted text-muted-foreground'}`}>
                                  {v.severity}
                                </span>
                              </div>
                            ))}
                          </div>
                        )}
                      </CardContent>
                    </Card>
                  );
                })}
              </div>
            )}

            {selectedFrameworkData && selectedFrameworkData.violations.length > 0 && (
              <div className="rounded-xl border bg-card">
                <div className="border-b px-5 py-4">
                  <h3 className="text-sm font-semibold">
                    {FRAMEWORK_METADATA[selectedFrameworkData.name]?.label ?? selectedFrameworkData.name} Violations
                  </h3>
                  <p className="text-xs text-muted-foreground">
                    {selectedFrameworkData.violations.length} violation{selectedFrameworkData.violations.length !== 1 ? 's' : ''} detected
                  </p>
                </div>
                <div className="divide-y">
                  {selectedFrameworkData.violations.map((v, idx) => (
                    <div key={`${v.policy_id}-${v.asset_id}-${idx}`} className="flex items-start justify-between gap-4 px-5 py-4">
                      <div className="min-w-0 space-y-1">
                        <p className="text-sm font-medium">{v.policy_name}</p>
                        <p className="text-xs text-muted-foreground">{v.description}</p>
                        <div className="flex flex-wrap items-center gap-2">
                          <Badge variant="outline" className="text-xs">{v.asset_name}</Badge>
                          <Badge variant="outline" className="text-xs capitalize">{v.asset_type.replace(/_/g, ' ')}</Badge>
                          <Badge variant="outline" className="text-xs capitalize">{v.classification}</Badge>
                          <span className="text-xs text-muted-foreground capitalize">
                            {v.enforcement.replace(/_/g, ' ')}
                          </span>
                        </div>
                      </div>
                      <span className={`inline-flex shrink-0 rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${SEVERITY_COLORS[v.severity] ?? 'bg-muted text-muted-foreground'}`}>
                        {v.severity}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </PermissionRedirect>
  );
}
