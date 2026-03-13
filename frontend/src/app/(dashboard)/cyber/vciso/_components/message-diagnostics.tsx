'use client';

import { useState } from 'react';
import { Loader2, Route, ScanSearch } from 'lucide-react';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Separator } from '@/components/ui/separator';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { apiGet } from '@/lib/api';
import { formatCompactNumber, parseApiError } from '@/lib/format';
import { API_ENDPOINTS } from '@/lib/constants';
import { cn, formatDateTime } from '@/lib/utils';
import type { VCISOConversationMessage, VCISOLLMAuditResponse } from '@/types/cyber';

interface MessageDiagnosticsProps {
  message: VCISOConversationMessage;
}

export function MessageDiagnostics({ message }: MessageDiagnosticsProps) {
  const [open, setOpen] = useState(false);
  const [audit, setAudit] = useState<VCISOLLMAuditResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const meta = message.meta;
  const engine = meta?.engine ?? message.engine ?? null;
  const canViewTrace = message.role === 'assistant' && (engine === 'llm' || engine === 'fallback');
  const hasVisibleMeta =
    Boolean(engine) ||
    Boolean(meta?.grounding) ||
    Boolean(meta?.tokens_used) ||
    Boolean(meta?.routing_reason) ||
    Boolean(meta?.reasoning_steps);

  if (!hasVisibleMeta && !canViewTrace) {
    return null;
  }

  async function handleOpen(nextOpen: boolean) {
    setOpen(nextOpen);
    if (!nextOpen || !canViewTrace || audit || isLoading) {
      return;
    }

    setIsLoading(true);
    setError(null);
    try {
      const response = await apiGet<VCISOLLMAuditResponse>(
        `${API_ENDPOINTS.CYBER_VCISO_LLM_AUDIT}/${message.id}`,
      );
      setAudit(response);
    } catch (err) {
      setError(parseApiError(err));
    } finally {
      setIsLoading(false);
    }
  }

  return (
    <>
      <div className="mt-3 flex flex-wrap items-center gap-2">
        {engine && (
          <Badge
            variant="outline"
            className={cn('rounded-full text-[10px] uppercase tracking-[0.12em]', engineColor(engine))}
          >
            {engineLabel(engine)}
          </Badge>
        )}
        {meta?.grounding && (
          <Badge variant="outline" className="rounded-full text-[10px]">
            Grounding: {meta.grounding}
          </Badge>
        )}
        {meta?.tokens_used ? (
          <Badge variant="outline" className="rounded-full text-[10px]">
            {formatCompactNumber(meta.tokens_used)} tokens
          </Badge>
        ) : null}
        {meta?.reasoning_steps ? (
          <Badge variant="outline" className="rounded-full text-[10px]">
            {meta.reasoning_steps} reasoning steps
          </Badge>
        ) : null}
        {canViewTrace && (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="h-6 rounded-full px-2 text-[10px]"
            onClick={() => void handleOpen(true)}
          >
            <ScanSearch className="mr-1 h-3 w-3" />
            View trace
          </Button>
        )}
      </div>
      {meta?.routing_reason && (
        <div className="mt-2 flex items-start gap-2 text-xs text-muted-foreground">
          <Route className="mt-0.5 h-3 w-3 shrink-0" />
          <span>Route: {humanize(meta.routing_reason)}</span>
        </div>
      )}

      <Sheet open={open} onOpenChange={(nextOpen) => void handleOpen(nextOpen)}>
        <SheetContent side="right" className="w-full overflow-y-auto sm:max-w-2xl">
          <SheetHeader>
            <SheetTitle>LLM Trace</SheetTitle>
            <SheetDescription>
              Inspect routing, token usage, tool calls, and reasoning for this assistant response.
            </SheetDescription>
          </SheetHeader>

          <div className="mt-6 space-y-6">
            <div className="grid gap-3 md:grid-cols-2">
              <SummaryCard title="Engine" value={engineLabel(engine ?? 'rule_based')} detail={meta?.routing_reason ? humanize(meta.routing_reason) : 'No routing reason recorded'} />
              <SummaryCard title="Grounding" value={meta?.grounding ?? audit?.grounding_result ?? '—'} detail={`Created ${formatDateTime(audit?.created_at ?? message.created_at)}`} />
              <SummaryCard title="Tokens" value={audit ? formatCompactNumber(audit.total_tokens) : meta?.tokens_used ? formatCompactNumber(meta.tokens_used) : '—'} detail={audit ? `${formatCompactNumber(audit.prompt_tokens)} prompt / ${formatCompactNumber(audit.completion_tokens)} completion` : 'Per-response token estimate'} />
              <SummaryCard title="Reasoning" value={audit ? String(audit.reasoning_trace.length) : meta?.reasoning_steps ? String(meta.reasoning_steps) : '—'} detail={audit ? `${audit.tool_calls.length} tool calls recorded` : 'Reasoning count from response metadata'} />
            </div>

            {isLoading && (
              <div className="flex items-center gap-3 rounded-2xl border bg-slate-50 px-4 py-3 text-sm text-muted-foreground">
                <Loader2 className="h-4 w-4 animate-spin" />
                Loading trace details...
              </div>
            )}

            {!isLoading && error && (
              <div className="rounded-2xl border border-dashed px-4 py-3 text-sm text-muted-foreground">
                {error}
              </div>
            )}

            {!isLoading && audit && (
              <>
                <div className="grid gap-3 md:grid-cols-3">
                  <SummaryCard title="Provider" value={audit.provider} detail={audit.model} />
                  <SummaryCard title="Routing" value={humanize(audit.engine_used)} detail={audit.routing_reason ? humanize(audit.routing_reason) : 'No routing reason recorded'} />
                  <SummaryCard title="Logged" value={formatDateTime(audit.created_at)} detail={`Message ${message.id.slice(0, 8)}`} />
                </div>

                <Card className="border-border/70">
                  <CardHeader className="pb-3">
                    <CardTitle className="text-base">Reasoning Trace</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    {audit.reasoning_trace.length === 0 ? (
                      <p className="text-sm text-muted-foreground">No explicit reasoning trace recorded.</p>
                    ) : (
                      audit.reasoning_trace.map((step) => (
                        <div key={`${step.step}-${step.action}`} className="rounded-2xl border bg-slate-50/80 p-3">
                          <div className="flex items-center justify-between gap-3">
                            <p className="text-sm font-semibold">
                              Step {step.step}: {humanize(step.action)}
                            </p>
                            {step.tool_names && step.tool_names.length > 0 && (
                              <Badge variant="outline" className="rounded-full text-[10px]">
                                {step.tool_names.length} tools
                              </Badge>
                            )}
                          </div>
                          <p className="mt-2 text-sm text-muted-foreground">{step.detail}</p>
                          {step.tool_names && step.tool_names.length > 0 && (
                            <div className="mt-3 flex flex-wrap gap-2">
                              {step.tool_names.map((toolName) => (
                                <Badge key={toolName} variant="outline" className="rounded-full text-[10px]">
                                  {toolName}
                                </Badge>
                              ))}
                            </div>
                          )}
                        </div>
                      ))
                    )}
                  </CardContent>
                </Card>

                <Card className="border-border/70">
                  <CardHeader className="pb-3">
                    <CardTitle className="text-base">Tool Calls</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    {audit.tool_calls.length === 0 ? (
                      <p className="text-sm text-muted-foreground">This response did not invoke any tools.</p>
                    ) : (
                      audit.tool_calls.map((toolCall, index) => (
                        <div key={`${toolCall.name}-${index}`} className="rounded-2xl border p-3">
                          <div className="flex flex-wrap items-center gap-2">
                            <p className="text-sm font-semibold">{toolCall.name}</p>
                            <Badge
                              variant="outline"
                              className={cn(
                                'rounded-full text-[10px]',
                                toolCall.success ? 'border-emerald-200 text-emerald-700' : 'border-rose-200 text-rose-700',
                              )}
                            >
                              {toolCall.success ? 'Success' : 'Failed'}
                            </Badge>
                            <Badge variant="outline" className="rounded-full text-[10px]">
                              {toolCall.latency_ms}ms
                            </Badge>
                          </div>
                          {Object.keys(toolCall.arguments ?? {}).length > 0 && (
                            <div className="mt-3 rounded-xl bg-slate-50 p-3">
                              <p className="mb-2 text-[11px] font-medium uppercase tracking-[0.12em] text-muted-foreground">
                                Arguments
                              </p>
                              <pre className="overflow-x-auto text-xs">
                                {JSON.stringify(toolCall.arguments, null, 2)}
                              </pre>
                            </div>
                          )}
                          {toolCall.result_summary && (
                            <>
                              <Separator className="my-3" />
                              <p className="text-sm text-muted-foreground">{toolCall.result_summary}</p>
                            </>
                          )}
                        </div>
                      ))
                    )}
                  </CardContent>
                </Card>
              </>
            )}
          </div>
        </SheetContent>
      </Sheet>
    </>
  );
}

function SummaryCard({
  title,
  value,
  detail,
}: {
  title: string;
  value: string;
  detail: string;
}) {
  return (
    <Card className="border-border/70">
      <CardHeader className="pb-2">
        <CardTitle className="text-sm text-muted-foreground">{title}</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-lg font-semibold">{value}</p>
        <p className="mt-1 text-xs text-muted-foreground">{detail}</p>
      </CardContent>
    </Card>
  );
}

function engineLabel(engine: string): string {
  switch (engine) {
    case 'llm':
      return 'LLM';
    case 'rule_based':
      return 'Deterministic';
    case 'fallback':
      return 'Fallback';
    default:
      return humanize(engine);
  }
}

function engineColor(engine: string): string {
  switch (engine) {
    case 'llm':
      return 'border-sky-200 text-sky-700';
    case 'fallback':
      return 'border-amber-200 text-amber-700';
    case 'rule_based':
      return 'border-slate-200 text-slate-700';
    default:
      return 'border-border text-foreground';
  }
}

function humanize(value: string): string {
  return value
    .replace(/[_-]+/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
}
