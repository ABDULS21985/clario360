'use client';

import { Bot, ShieldAlert, User2 } from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
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
          <div className="whitespace-pre-wrap text-sm leading-6">{message.content}</div>
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
