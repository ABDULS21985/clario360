import {
  Activity,
  AlertTriangle,
  BarChart3,
  CheckCircle2,
  Cloud,
  Database,
  FileQuestion,
  FileSpreadsheet,
  Flame,
  GitBranch,
  GitCommit,
  Globe,
  HardDrive,
  type LucideIcon,
  ShieldAlert,
  Warehouse,
  Waves,
  Zap,
} from 'lucide-react';
import { formatBytes, formatCompactNumber, formatDate, formatDateTime, formatRelativeTime, formatDuration } from '@/lib/format';
import { titleCase } from '@/lib/format';
import type {
  DataClassification,
  DataSourceType,
  DataSuiteDashboard,
  DiscoveredColumn,
  JsonValue,
  PipelineRun,
} from '@/lib/data-suite/types';

export interface BadgeVisual {
  label: string;
  className: string;
}

export interface SourceTypeVisual {
  label: string;
  icon: LucideIcon;
  accentClass: string;
}

export const sourceTypeVisuals: Record<DataSourceType, SourceTypeVisual> = {
  postgresql: { label: 'PostgreSQL', icon: Database, accentClass: 'text-sky-600' },
  mysql: { label: 'MySQL', icon: Database, accentClass: 'text-orange-500' },
  mssql: { label: 'MS SQL', icon: Database, accentClass: 'text-indigo-600' },
  api: { label: 'REST API', icon: Globe, accentClass: 'text-fuchsia-600' },
  csv: { label: 'CSV / TSV', icon: FileSpreadsheet, accentClass: 'text-emerald-600' },
  s3: { label: 'S3 / MinIO', icon: Cloud, accentClass: 'text-amber-600' },
  clickhouse: { label: 'ClickHouse', icon: BarChart3, accentClass: 'text-rose-600' },
  impala: { label: 'Apache Impala', icon: Zap, accentClass: 'text-violet-600' },
  hive: { label: 'Apache Hive', icon: Warehouse, accentClass: 'text-yellow-700' },
  hdfs: { label: 'HDFS', icon: HardDrive, accentClass: 'text-slate-700' },
  spark: { label: 'Apache Spark', icon: Flame, accentClass: 'text-orange-600' },
  dagster: { label: 'Dagster', icon: GitBranch, accentClass: 'text-cyan-700' },
  dolt: { label: 'Dolt', icon: GitCommit, accentClass: 'text-emerald-700' },
  stream: { label: 'Streaming', icon: Activity, accentClass: 'text-teal-600' },
};

export const classificationBadgeMap: Record<DataClassification, BadgeVisual> = {
  public: {
    label: 'Public',
    className: 'border-emerald-200 bg-emerald-50 text-emerald-700',
  },
  internal: {
    label: 'Internal',
    className: 'border-sky-200 bg-sky-50 text-sky-700',
  },
  confidential: {
    label: 'Confidential',
    className: 'border-amber-200 bg-amber-50 text-amber-700',
  },
  restricted: {
    label: 'Restricted',
    className: 'border-rose-200 bg-rose-50 text-rose-700',
  },
};

export const contradictionTypeVisuals: Record<string, BadgeVisual> = {
  logical: { label: 'Logical', className: 'border-fuchsia-200 bg-fuchsia-50 text-fuchsia-700' },
  semantic: { label: 'Semantic', className: 'border-sky-200 bg-sky-50 text-sky-700' },
  temporal: { label: 'Temporal', className: 'border-amber-200 bg-amber-50 text-amber-700' },
  analytical: { label: 'Analytical', className: 'border-rose-200 bg-rose-50 text-rose-700' },
};

export const qualitySeverityVisuals: Record<string, { label: string; className: string; icon: LucideIcon }> = {
  critical: { label: 'Critical', className: 'border-rose-300 bg-rose-50 text-rose-700', icon: ShieldAlert },
  high: { label: 'High', className: 'border-red-200 bg-red-50 text-red-700', icon: AlertTriangle },
  medium: { label: 'Medium', className: 'border-amber-200 bg-amber-50 text-amber-700', icon: Waves },
  low: { label: 'Low', className: 'border-slate-200 bg-slate-50 text-slate-700', icon: CheckCircle2 },
};

export const dashboardKpiIcons = {
  total_sources: Database,
  active_pipelines: GitBranch,
  quality_score: CheckCircle2,
  open_contradictions: AlertTriangle,
  dark_data_assets: FileQuestion,
};

export function getClassificationBadge(value?: string | null): BadgeVisual {
  if (!value) {
    return { label: 'Unknown', className: 'border-slate-200 bg-slate-50 text-slate-700' };
  }
  const lowered = value.toLowerCase() as DataClassification;
  return classificationBadgeMap[lowered] ?? {
    label: titleCase(value),
    className: 'border-slate-200 bg-slate-50 text-slate-700',
  };
}

export function getSourceTypeVisual(value: string): SourceTypeVisual {
  const lowered = value.toLowerCase() as DataSourceType;
  return sourceTypeVisuals[lowered] ?? {
    label: titleCase(value),
    icon: Database,
    accentClass: 'text-slate-600',
  };
}

export function getStatusTone(status: string): string {
  switch (status) {
    case 'active':
    case 'completed':
    case 'success':
    case 'passed':
    case 'governed':
    case 'resolved':
      return 'bg-emerald-500';
    case 'syncing':
    case 'running':
    case 'investigating':
    case 'processing':
      return 'bg-sky-500';
    case 'pending_test':
    case 'warning':
    case 'paused':
    case 'accepted':
    case 'under_review':
      return 'bg-amber-500';
    case 'error':
    case 'failed':
    case 'critical':
    case 'false_positive':
    case 'scheduled_deletion':
      return 'bg-rose-500';
    default:
      return 'bg-slate-400';
  }
}

export function formatMaybeBytes(value?: number | null): string {
  if (value === undefined || value === null) {
    return '—';
  }
  return formatBytes(value);
}

export function formatMaybeCompact(value?: number | null): string {
  if (value === undefined || value === null) {
    return '—';
  }
  return formatCompactNumber(value);
}

export function formatMaybeDate(value?: string | null): string {
  return value ? formatDate(value) : '—';
}

export function formatMaybeDateTime(value?: string | null): string {
  return value ? formatDateTime(value) : '—';
}

export function formatMaybeRelative(value?: string | null): string {
  return value ? formatRelativeTime(value) : '—';
}

export function formatMaybeDurationMs(value?: number | null): string {
  if (value === undefined || value === null) {
    return '—';
  }
  return formatDuration(Math.round(value / 1000));
}

export function normalizeConnectionHost(config?: Record<string, JsonValue> | null): string {
  if (!config) {
    return 'unknown';
  }

  const host = config.host;
  if (typeof host === 'string' && host.trim() !== '') {
    return host;
  }

  const graphQlUrl = config.graphql_url;
  if (typeof graphQlUrl === 'string' && graphQlUrl.trim() !== '') {
    try {
      return new URL(graphQlUrl).hostname;
    } catch {
      return graphQlUrl;
    }
  }

  const baseUrl = config.base_url;
  if (typeof baseUrl === 'string' && baseUrl.trim() !== '') {
    try {
      return new URL(baseUrl).hostname;
    } catch {
      return baseUrl;
    }
  }

  const bucket = config.bucket;
  if (typeof bucket === 'string' && bucket.trim() !== '') {
    return bucket;
  }

  const nameNodes = config.name_nodes;
  if (Array.isArray(nameNodes) && typeof nameNodes[0] === 'string' && nameNodes[0].trim() !== '') {
    return nameNodes[0];
  }

  const restConfig = config.rest;
  if (restConfig && typeof restConfig === 'object' && !Array.isArray(restConfig)) {
    const masterUrl = (restConfig as Record<string, JsonValue>).master_url;
    if (typeof masterUrl === 'string' && masterUrl.trim() !== '') {
      try {
        return new URL(masterUrl).hostname;
      } catch {
        return masterUrl;
      }
    }
  }

  return 'unknown';
}

export function deriveSourceName(type: DataSourceType, connectionConfig?: Record<string, JsonValue> | null): string {
  const host = slugify(normalizeConnectionHost(connectionConfig));
  const db = slugify(
    typeof connectionConfig?.database === 'string'
      ? connectionConfig.database
      : typeof connectionConfig?.graphql_url === 'string'
        ? 'dagster'
        : typeof connectionConfig?.branch === 'string'
          ? connectionConfig.branch
      : typeof connectionConfig?.file_path === 'string'
        ? connectionConfig.file_path.split('/').pop() ?? 'data'
        : 'source',
  );
  return `${type}_${host}_${db}`.replace(/_+/g, '_').replace(/^_|_$/g, '');
}

export function deriveModelName(tableName: string): string {
  return tableName
    .replace(/\./g, '_')
    .replace(/[^a-zA-Z0-9_]/g, '_')
    .replace(/s$/, '')
    .replace(/_+/g, '_')
    .toLowerCase();
}

export function humanizeCronOrFrequency(value?: string | null): string {
  if (!value) {
    return 'Manual only';
  }
  switch (value) {
    case '@hourly':
    case '0 * * * *':
      return 'Every hour';
    case '0 */6 * * *':
      return 'Every 6 hours';
    case '0 */12 * * *':
      return 'Every 12 hours';
    case '0 0 * * *':
      return 'Daily';
    case '0 0 * * 0':
      return 'Weekly';
    default:
      return value;
  }
}

export function maskPiiValue(value: JsonValue, piiType?: string | null): string {
  const stringValue = `${value ?? ''}`;
  if (!stringValue) {
    return '—';
  }

  switch ((piiType ?? '').toLowerCase()) {
    case 'email': {
      const [userPart, domainPart] = stringValue.split('@');
      if (!userPart || !domainPart) {
        return maskWithVisibleEdges(stringValue, 1, 0);
      }
      const [domainName, ...tldParts] = domainPart.split('.');
      const visibleDomain = maskWithVisibleEdges(domainName, 1, 0);
      const visibleUser = maskWithVisibleEdges(userPart, 1, 0);
      return `${visibleUser}@${visibleDomain}${tldParts.length > 0 ? `.${tldParts.join('.')}` : ''}`;
    }
    case 'phone':
      return stringValue.replace(/\d(?=\d{4})/g, '*');
    case 'ssn':
      return stringValue.replace(/\d(?=\d{2})/g, '*');
    case 'name':
      return stringValue
        .split(/\s+/)
        .map((part) => maskWithVisibleEdges(part, 1, 0))
        .join(' ');
    default:
      return maskWithVisibleEdges(stringValue, 1, 1);
  }
}

export function maskColumnSample(column: DiscoveredColumn, canViewPii: boolean): string[] {
  if (canViewPii || !column.inferred_pii) {
    return column.sample_values ?? [];
  }
  return (column.sample_values ?? []).map((value) => maskPiiValue(value, column.inferred_pii_type));
}

export function getColumnPiiType(column: DiscoveredColumn): string | null {
  if (!column.inferred_pii) {
    return null;
  }
  return column.inferred_pii_type || 'pii';
}

export function buildPipelineTrendSeries(dashboard: DataSuiteDashboard): Array<Record<string, string | number>> {
  const failedCount = dashboard.kpis.failed_pipelines_24h;
  return dashboard.pipeline_trend_30d.map((point, index) => {
    const completed = Math.max(Math.round(point.value), 0);
    const failureEstimate = index === dashboard.pipeline_trend_30d.length - 1 ? failedCount : 0;
    const failed = Math.min(completed, failureEstimate);
    return {
      day: point.day,
      success: Math.max(completed - failed, 0),
      failed,
      cancelled: 0,
    };
  });
}

export function buildSourceStatusChartRows(dashboard: DataSuiteDashboard): Array<Record<string, string | number>> {
  const totalStatusCount = Object.values(dashboard.sources_by_status).reduce((sum, value) => sum + value, 0);
  const types = Object.entries(dashboard.sources_by_type);

  return types.map(([type, count]) => {
    const row: Record<string, string | number> = { type: titleCase(type) };
    for (const status of ['active', 'inactive', 'error', 'syncing']) {
      const share = totalStatusCount > 0 ? (dashboard.sources_by_status[status] ?? 0) / totalStatusCount : 0;
      row[status] = Math.round(count * share);
    }
    return row;
  });
}

export function getRunCompletionLabel(run: PipelineRun): string {
  if (run.completed_at) {
    return formatRelativeTime(run.completed_at);
  }
  if (run.started_at) {
    return formatRelativeTime(run.started_at);
  }
  return '—';
}

export function safeJsonString(value: JsonValue): string {
  if (typeof value === 'string') {
    return value;
  }
  try {
    return JSON.stringify(value);
  } catch {
    return '';
  }
}

function slugify(value: string): string {
  return value
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '');
}

function maskWithVisibleEdges(value: string, visibleStart: number, visibleEnd: number): string {
  if (value.length <= visibleStart + visibleEnd) {
    return '*'.repeat(Math.max(value.length, 1));
  }
  const start = value.slice(0, visibleStart);
  const end = visibleEnd > 0 ? value.slice(-visibleEnd) : '';
  return `${start}${'*'.repeat(Math.max(value.length - visibleStart - visibleEnd, 3))}${end}`;
}
