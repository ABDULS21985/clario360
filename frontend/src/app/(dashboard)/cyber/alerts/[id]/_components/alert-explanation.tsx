'use client';

import type { ReactNode } from 'react';
import { AlertCircle, CheckCircle2, Lightbulb, ShieldAlert, Sparkles } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { alertConfidencePercent } from '@/lib/cyber-alerts';
import type { CyberAlert } from '@/types/cyber';

import { ConfidenceGauge } from './confidence-gauge';

interface AlertExplanationProps {
  alert: CyberAlert;
}

export function AlertExplanation({ alert }: AlertExplanationProps) {
  const explanation = alert.explanation;

  return (
    <div className="space-y-6">
      <section className="rounded-[26px] border bg-[linear-gradient(140deg,rgba(248,250,252,0.95),rgba(236,253,245,0.7))] p-5 shadow-sm">
        <div className="flex flex-col gap-5 lg:flex-row lg:items-start">
          <ConfidenceGauge score={alertConfidencePercent(alert.confidence_score)} size="md" />
          <div className="space-y-4">
            <div>
              <p className="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Summary
              </p>
              <p className="mt-2 text-sm leading-7 text-slate-700">
                {explanation.summary || 'No AI summary was generated for this alert.'}
              </p>
            </div>
            <div>
              <p className="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                Why This Matters
              </p>
              <p className="mt-2 text-sm leading-7 text-slate-700">
                {explanation.reason || 'No reason was supplied.'}
              </p>
            </div>
          </div>
        </div>
      </section>

      <Section icon={CheckCircle2} title="Matched Conditions">
        {(explanation.matched_conditions?.length ?? 0) > 0 ? (
          <div className="flex flex-wrap gap-2">
            {explanation.matched_conditions.map((condition) => (
              <Badge key={condition} variant="secondary">
                {condition}
              </Badge>
            ))}
          </div>
        ) : (
          <EmptyMessage message="No matched conditions were recorded." />
        )}
      </Section>

      <Section icon={Sparkles} title="Confidence Factors">
        {(explanation.confidence_factors?.length ?? 0) > 0 ? (
          <div className="space-y-3">
            {explanation.confidence_factors.map((factor, index) => (
              <div key={`${factor.factor}-${index}`} className="rounded-2xl border bg-background px-4 py-3">
                <div className="flex items-center justify-between gap-3">
                  <p className="text-sm font-medium text-slate-900">{factor.factor}</p>
                  <Badge variant={factor.impact >= 0 ? 'secondary' : 'outline'}>
                    {(factor.impact * 100).toFixed(0)}%
                  </Badge>
                </div>
                <p className="mt-1 text-sm text-muted-foreground">{factor.description}</p>
              </div>
            ))}
          </div>
        ) : (
          <EmptyMessage message="No confidence factors were supplied." />
        )}
      </Section>

      <Section icon={Lightbulb} title="Recommended Actions">
        {(explanation.recommended_actions?.length ?? 0) > 0 ? (
          <div className="space-y-3">
            {explanation.recommended_actions.map((action, index) => (
              <div key={`${action}-${index}`} className="flex gap-3 rounded-2xl border bg-background px-4 py-3">
                <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-emerald-100 text-xs font-semibold text-emerald-700">
                  {index + 1}
                </span>
                <p className="text-sm text-slate-700">{action}</p>
              </div>
            ))}
          </div>
        ) : (
          <EmptyMessage message="No recommended actions were generated." />
        )}
      </Section>

      <Section icon={ShieldAlert} title="False Positive Indicators">
        {(explanation.false_positive_indicators?.length ?? 0) > 0 ? (
          <div className="space-y-2">
            {explanation.false_positive_indicators.map((indicator, index) => (
              <div key={`${indicator}-${index}`} className="rounded-2xl border border-purple-200 bg-purple-50/70 px-4 py-3 text-sm text-purple-900">
                {indicator}
              </div>
            ))}
          </div>
        ) : (
          <EmptyMessage message="No false-positive indicators were recorded." />
        )}
      </Section>

      <Section icon={AlertCircle} title="Indicator Matches">
        {(explanation.indicator_matches?.length ?? 0) > 0 ? (
          <div className="grid gap-3 md:grid-cols-2">
            {explanation.indicator_matches?.map((match, index) => (
              <div key={`${match.value}-${index}`} className="rounded-2xl border bg-background px-4 py-3">
                <div className="flex flex-wrap items-center gap-2">
                  <Badge variant="outline">{match.type}</Badge>
                  <Badge variant="secondary">{Math.round(match.confidence * 100)}%</Badge>
                  <Badge variant="secondary">{match.source}</Badge>
                </div>
                <p className="mt-3 break-all font-mono text-xs text-slate-900">{match.value}</p>
                {match.field && (
                  <p className="mt-2 text-xs text-muted-foreground">
                    Matched field: <span className="font-mono">{match.field}</span>
                  </p>
                )}
              </div>
            ))}
          </div>
        ) : (
          <EmptyMessage message="No supporting indicator matches were attached to this alert." />
        )}
      </Section>
    </div>
  );
}

function Section({
  icon: Icon,
  title,
  children,
}: {
  icon: typeof AlertCircle;
  title: string;
  children: ReactNode;
}) {
  return (
    <section className="rounded-[26px] border bg-card p-5 shadow-sm">
      <div className="mb-4 flex items-center gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-2xl border border-slate-200 bg-slate-50 text-slate-700">
          <Icon className="h-4 w-4" />
        </div>
        <div>
          <p className="text-xs font-semibold uppercase tracking-[0.2em] text-muted-foreground">
            AI Explanation
          </p>
          <h2 className="text-lg font-semibold tracking-[-0.03em] text-slate-950">{title}</h2>
        </div>
      </div>
      {children}
    </section>
  );
}

function EmptyMessage({ message }: { message: string }) {
  return <p className="text-sm text-muted-foreground">{message}</p>;
}
