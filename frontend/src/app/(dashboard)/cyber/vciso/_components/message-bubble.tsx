'use client';

import { Bot, ShieldAlert, User2 } from 'lucide-react';
import ReactMarkdown from 'react-markdown';

import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { BarChart } from '@/components/shared/charts/bar-chart';
import { LineChart } from '@/components/shared/charts/line-chart';
import { PieChart } from '@/components/shared/charts/pie-chart';
import { cn, formatDateTime } from '@/lib/utils';
import type { VCISOConversationMessage, VCISOSuggestedAction } from '@/types/cyber';

interface MessageBubbleProps {
  message: VCISOConversationMessage;
  onAction: (action: VCISOSuggestedAction) => void;
}

export function MessageBubble({ message, onAction }: MessageBubbleProps) {
  const isUser = message.role === 'user';

  return (
    <div className={cn('flex gap-3', isUser ? 'justify-end' : 'justify-start')}>
      {!isUser && (
        <div className="mt-1 flex h-9 w-9 shrink-0 items-center justify-center rounded-2xl bg-slate-900 text-white">
          {message.role === 'system' ? <ShieldAlert className="h-4 w-4" /> : <Bot className="h-4 w-4" />}
        </div>
      )}
      <div className={cn('max-w-[90%] space-y-2', isUser && 'items-end')}>
        <div
          className={cn(
            'rounded-3xl border px-4 py-3 shadow-sm',
            isUser ? 'border-primary/20 bg-primary text-primary-foreground' : 'border-border bg-white',
          )}
        >
          <div className="text-sm leading-6 [&>*:first-child]:mt-0 [&>*:last-child]:mb-0">
            {isUser ? (
              <span className="whitespace-pre-wrap">{message.content}</span>
            ) : (
              <ReactMarkdown
                components={{
                  p: ({ children }) => <p className="mb-2 last:mb-0">{children}</p>,
                  strong: ({ children }) => <strong className="font-semibold">{children}</strong>,
                  ul: ({ children }) => <ul className="mb-2 ml-4 list-disc space-y-1 last:mb-0">{children}</ul>,
                  ol: ({ children }) => <ol className="mb-2 ml-4 list-decimal space-y-1 last:mb-0">{children}</ol>,
                  li: ({ children }) => <li>{children}</li>,
                  code: ({ children }) => (
                    <code className="rounded bg-slate-100 px-1.5 py-0.5 text-xs font-mono">{children}</code>
                  ),
                  pre: ({ children }) => (
                    <pre className="mb-2 overflow-x-auto rounded-lg bg-slate-100 p-3 text-xs last:mb-0">{children}</pre>
                  ),
                  h1: ({ children }) => <h3 className="mb-1 mt-3 text-base font-semibold first:mt-0">{children}</h3>,
                  h2: ({ children }) => <h3 className="mb-1 mt-3 text-base font-semibold first:mt-0">{children}</h3>,
                  h3: ({ children }) => <h4 className="mb-1 mt-2 text-sm font-semibold first:mt-0">{children}</h4>,
                  a: ({ href, children }) => (
                    <a href={href} className="text-primary underline hover:no-underline" target="_blank" rel="noopener noreferrer">{children}</a>
                  ),
                  blockquote: ({ children }) => (
                    <blockquote className="mb-2 border-l-2 border-slate-300 pl-3 text-muted-foreground last:mb-0">{children}</blockquote>
                  ),
                }}
              >
                {message.content}
              </ReactMarkdown>
            )}
          </div>
          {!isUser && renderStructuredContent(message)}
        </div>
        {!isUser && message.actions.length > 0 && (
          <div className="flex flex-wrap gap-2">
            {message.actions.slice(0, 3).map((action) => (
              <Button
                key={`${message.id}-${action.label}`}
                type="button"
                size="sm"
                variant={action.type === 'navigate' ? 'default' : 'outline'}
                className="rounded-full"
                onClick={() => onAction(action)}
              >
                {action.label}
              </Button>
            ))}
          </div>
        )}
        <div className={cn('flex items-center gap-2 px-1 text-[11px] text-muted-foreground', isUser && 'justify-end')}>
          {isUser && <User2 className="h-3 w-3" />}
          <span>{formatDateTime(message.created_at)}</span>
          {message.intent && <Badge variant="outline" className="rounded-full text-[10px]">{message.intent}</Badge>}
        </div>
      </div>
    </div>
  );
}

function renderStructuredContent(message: VCISOConversationMessage) {
  const result = message.tool_result;
  if (!result || typeof result !== 'object') {
    return null;
  }

  switch (message.response_type) {
    case 'kpi':
      return renderKPI(result as Record<string, unknown>);
    case 'list':
      return renderList(result as Record<string, unknown>);
    case 'table':
      return renderTable(result as Record<string, unknown>);
    case 'chart':
      return renderChart(result as Record<string, unknown>);
    case 'dashboard':
      return renderDashboard(result as Record<string, unknown>);
    case 'investigation':
      return renderInvestigation(result as Record<string, unknown>);
    default:
      return null;
  }
}

function renderKPI(result: Record<string, unknown>) {
  const score = getNumber(result.score);
  const grade = getString(result.grade);
  const components = Array.isArray(result.components) ? result.components : [];

  return (
    <div className="mt-4 rounded-2xl border bg-slate-50 p-4">
      <div className="flex items-end justify-between gap-3">
        <div>
          <p className="text-xs uppercase tracking-[0.16em] text-muted-foreground">Current Score</p>
          <div className="mt-1 flex items-end gap-2">
            <span className="text-3xl font-semibold">{score ?? '—'}</span>
            {grade && <Badge className="mb-1 rounded-full">{grade}</Badge>}
          </div>
        </div>
      </div>
      {components.length > 0 && (
        <div className="mt-4 space-y-2">
          {components.slice(0, 4).map((item, index) => {
            if (!item || typeof item !== 'object') {
              return null;
            }
            const name = getString((item as Record<string, unknown>).name) ?? `Component ${index + 1}`;
            const value = getNumber((item as Record<string, unknown>).score);
            return (
              <div key={`${name}-${index}`} className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">{name}</span>
                <span className="font-medium">{value ?? '—'}</span>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

function renderList(result: Record<string, unknown>) {
  const items = extractListItems(result).slice(0, 5);
  if (items.length === 0) {
    return null;
  }
  return (
    <div className="mt-4 rounded-2xl border bg-slate-50 p-4">
      <p className="mb-3 text-xs uppercase tracking-[0.16em] text-muted-foreground">Highlights</p>
      <div className="space-y-2">
        {items.map((item, index) => (
          <div key={`${item}-${index}`} className="text-sm">
            <span className="mr-2 text-muted-foreground">{index + 1}.</span>
            <span>{item}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function renderDashboard(result: Record<string, unknown>) {
  const widgets = Array.isArray(result.widgets) ? result.widgets.length : 0;
  const dashboardId = getString(result.dashboard_id);
  return (
    <div className="mt-4 rounded-2xl border bg-slate-50 p-4">
      <p className="text-xs uppercase tracking-[0.16em] text-muted-foreground">Dashboard Created</p>
      <p className="mt-2 text-sm">Widgets: {widgets}</p>
      {dashboardId && <p className="mt-1 text-xs text-muted-foreground">ID: {dashboardId}</p>}
    </div>
  );
}

function renderInvestigation(result: Record<string, unknown>) {
  const alert = result.alert;
  if (!alert || typeof alert !== 'object') {
    return null;
  }
  const title = getString((alert as Record<string, unknown>).title);
  const severity = getString((alert as Record<string, unknown>).severity);
  const status = getString((alert as Record<string, unknown>).status);
  return (
    <div className="mt-4 rounded-2xl border bg-slate-50 p-4">
      <p className="text-xs uppercase tracking-[0.16em] text-muted-foreground">Investigation Snapshot</p>
      <p className="mt-2 text-sm font-semibold">{title ?? 'Alert investigation'}</p>
      <div className="mt-2 flex flex-wrap gap-2">
        {severity && <Badge variant="secondary" className="rounded-full">{severity}</Badge>}
        {status && <Badge variant="outline" className="rounded-full">{status}</Badge>}
      </div>
    </div>
  );
}

// ── Table renderer ──────────────────────────────────────────────────────────
// Handles asset_lookup (asset + vulnerabilities[] + alerts[]) and generic
// objects with arrays of records.

function renderTable(result: Record<string, unknown>) {
  const sections: React.ReactNode[] = [];

  // 1. Render the primary entity as key-value pairs (e.g., asset details)
  const primaryObj = findPrimaryObject(result);
  if (primaryObj) {
    const entries = Object.entries(primaryObj).filter(
      ([, v]) => v !== null && v !== undefined && typeof v !== 'object',
    );
    if (entries.length > 0) {
      sections.push(
        <div key="primary" className="overflow-hidden rounded-lg border">
          <table className="w-full text-xs">
            <tbody>
              {entries.slice(0, 12).map(([key, value]) => (
                <tr key={key} className="border-b last:border-0">
                  <td className="bg-slate-50 px-3 py-1.5 font-medium text-muted-foreground capitalize whitespace-nowrap">
                    {formatKey(key)}
                  </td>
                  <td className="px-3 py-1.5 break-all">{String(value)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>,
      );
    }
  }

  // 2. Render each array in the result as a data table (vulnerabilities, alerts, etc.)
  for (const [key, value] of Object.entries(result)) {
    if (!Array.isArray(value) || value.length === 0) continue;
    const first = value[0];
    if (!first || typeof first !== 'object') continue;

    const records = value as Array<Record<string, unknown>>;
    const columns = Object.keys(first).filter(
      (k) => typeof first[k] !== 'object' || first[k] === null,
    ).slice(0, 6);

    if (columns.length === 0) continue;

    sections.push(
      <div key={key}>
        <p className="mb-1.5 text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">
          {formatKey(key)}
        </p>
        <div className="overflow-x-auto rounded-lg border">
          <table className="w-full text-xs">
            <thead className="border-b bg-slate-50">
              <tr>
                {columns.map((col) => (
                  <th key={col} className="px-3 py-1.5 text-left font-medium text-muted-foreground capitalize whitespace-nowrap">
                    {formatKey(col)}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {records.slice(0, 10).map((row, idx) => (
                <tr key={idx} className="border-b last:border-0">
                  {columns.map((col) => (
                    <td key={col} className="px-3 py-1.5 whitespace-nowrap max-w-[200px] truncate">
                      {renderCellValue(row[col])}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
          {records.length > 10 && (
            <p className="px-3 py-1.5 text-xs text-muted-foreground border-t">
              +{records.length - 10} more rows
            </p>
          )}
        </div>
      </div>,
    );
  }

  if (sections.length === 0) return null;

  return (
    <div className="mt-4 space-y-3 rounded-2xl border bg-slate-50 p-4">
      <p className="text-xs uppercase tracking-[0.16em] text-muted-foreground">Details</p>
      {sections}
    </div>
  );
}

function findPrimaryObject(result: Record<string, unknown>): Record<string, unknown> | null {
  // Look for common primary entity keys (asset, entity, item, profile)
  for (const key of ['asset', 'entity', 'item', 'profile', 'source', 'rule']) {
    const val = result[key];
    if (val && typeof val === 'object' && !Array.isArray(val)) {
      return val as Record<string, unknown>;
    }
  }
  return null;
}

function formatKey(key: string): string {
  return key.replace(/_/g, ' ').replace(/([a-z])([A-Z])/g, '$1 $2');
}

function renderCellValue(value: unknown): string {
  if (value === null || value === undefined) return '—';
  if (typeof value === 'boolean') return value ? 'Yes' : 'No';
  if (typeof value === 'number') return value.toLocaleString();
  const str = String(value);
  // Format ISO timestamps inline
  if (/^\d{4}-\d{2}-\d{2}T/.test(str)) {
    try {
      return new Date(str).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
    } catch {
      return str;
    }
  }
  return str;
}

// ── Chart renderer ──────────────────────────────────────────────────────────
// Handles trend_analysis (risk_trend[]) and mitre_coverage (coverage_percent + gaps[])

const CHART_COLORS = [
  'hsl(142, 71%, 45%)', // green
  'hsl(217, 91%, 60%)', // blue
  'hsl(24, 95%, 53%)',  // orange
  'hsl(262, 83%, 58%)', // purple
  'hsl(350, 89%, 60%)', // red
  'hsl(186, 72%, 43%)', // teal
];

function renderChart(result: Record<string, unknown>) {
  // 1. Time-series: trend_analysis returns { risk_trend: [...] }
  const riskTrend = result.risk_trend;
  if (Array.isArray(riskTrend) && riskTrend.length > 0) {
    return renderTimeSeriesChart(riskTrend as Array<Record<string, unknown>>);
  }

  // 2. Coverage: mitre_coverage returns { coverage_percent, covered, total, gaps }
  if (typeof result.coverage_percent === 'number') {
    return renderCoverageChart(result);
  }

  // 3. Distribution: look for any array with { name, count|value } shape
  for (const [key, value] of Object.entries(result)) {
    if (!Array.isArray(value) || value.length === 0) continue;
    const first = value[0] as Record<string, unknown> | null;
    if (!first || typeof first !== 'object') continue;
    if (('name' in first) && ('count' in first || 'value' in first)) {
      return renderDistributionChart(key, value as Array<Record<string, unknown>>);
    }
  }

  return null;
}

function renderTimeSeriesChart(data: Array<Record<string, unknown>>) {
  // Detect numeric keys (skip time/date keys)
  const first = data[0];
  const timeKey = Object.keys(first).find(
    (k) => k === 'time' || k === 'date' || k === 'timestamp' || k.endsWith('_at'),
  ) ?? 'time';
  const numericKeys = Object.keys(first).filter(
    (k) => k !== timeKey && typeof first[k] === 'number',
  );

  if (numericKeys.length === 0) return null;

  // Format time values for display
  const formatted = data.map((point) => {
    const raw = point[timeKey];
    let label = String(raw);
    if (typeof raw === 'string' && /^\d{4}-\d{2}-\d{2}/.test(raw)) {
      try {
        label = new Date(raw).toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
      } catch { /* keep raw */ }
    }
    return { ...point, [timeKey]: label };
  });

  const yKeys = numericKeys.slice(0, 4).map((key, i) => ({
    key,
    label: formatKey(key),
    color: CHART_COLORS[i % CHART_COLORS.length],
  }));

  return (
    <div className="mt-4 rounded-2xl border bg-slate-50 p-4">
      <p className="mb-2 text-xs uppercase tracking-[0.16em] text-muted-foreground">Trend</p>
      <LineChart
        data={formatted}
        xKey={timeKey}
        yKeys={yKeys}
        height={200}
        showGrid={false}
        showLegend={yKeys.length > 1}
      />
    </div>
  );
}

function renderCoverageChart(result: Record<string, unknown>) {
  const covered = getNumber(result.covered) ?? 0;
  const total = getNumber(result.total) ?? 0;
  const pct = getNumber(result.coverage_percent) ?? 0;
  const gaps = Array.isArray(result.gaps) ? result.gaps as Array<Record<string, unknown>> : [];

  const pieData = [
    { name: 'Covered', value: covered, color: 'hsl(142, 71%, 45%)' },
    { name: 'Gaps', value: Math.max(0, total - covered), color: 'hsl(0, 84%, 60%)' },
  ];

  return (
    <div className="mt-4 rounded-2xl border bg-slate-50 p-4">
      <p className="mb-2 text-xs uppercase tracking-[0.16em] text-muted-foreground">Coverage</p>
      <PieChart
        data={pieData}
        height={180}
        innerRadius={45}
        outerRadius={70}
        centerValue={`${Math.round(pct)}%`}
        centerLabel="coverage"
        showLegend
      />
      {gaps.length > 0 && (
        <div className="mt-3">
          <p className="mb-1 text-xs font-medium text-muted-foreground">Top Gaps</p>
          <div className="space-y-1">
            {gaps.slice(0, 5).map((gap, i) => (
              <div key={i} className="flex items-center gap-2 text-xs">
                <Badge variant="outline" className="rounded-full text-[10px] shrink-0">
                  {getString(gap.id) ?? `#${i + 1}`}
                </Badge>
                <span className="truncate">{getString(gap.name) ?? 'Unknown'}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function renderDistributionChart(title: string, data: Array<Record<string, unknown>>) {
  const items = data.slice(0, 8).map((item, i) => ({
    name: getString(item.name) ?? `Item ${i + 1}`,
    value: getNumber(item.count) ?? getNumber(item.value) ?? 0,
    color: CHART_COLORS[i % CHART_COLORS.length],
  }));

  if (items.every((d) => d.value === 0)) return null;

  return (
    <div className="mt-4 rounded-2xl border bg-slate-50 p-4">
      <p className="mb-2 text-xs uppercase tracking-[0.16em] text-muted-foreground">{formatKey(title)}</p>
      <BarChart
        data={items}
        xKey="name"
        yKeys={[{ key: 'value', label: 'Count', color: CHART_COLORS[1] }]}
        height={180}
        showGrid={false}
        showLegend={false}
        barRadius={4}
      />
    </div>
  );
}

function extractListItems(result: Record<string, unknown>): string[] {
  const collections = [
    result.alerts,
    result.items,
    result.pipelines,
    result.priorities,
  ];
  for (const collection of collections) {
    if (!Array.isArray(collection)) {
      continue;
    }
    return collection
      .map((item) => {
        if (!item || typeof item !== 'object') {
          return null;
        }
        const record = item as Record<string, unknown>;
        return getString(record.title) ?? getString(record.name) ?? getString(record.detail);
      })
      .filter((item): item is string => Boolean(item));
  }
  return [];
}

function getString(value: unknown): string | null {
  return typeof value === 'string' && value.trim().length > 0 ? value : null;
}

function getNumber(value: unknown): number | null {
  return typeof value === 'number' ? value : null;
}
