'use client';

import type { DetectionRule, SigmaRuleContent } from '@/types/cyber';
import { serializeRuleContent, stringifySigmaContent } from '@/lib/cyber-rules';

import { RuleSigmaMonaco } from '../../_components/rule-sigma-monaco';

function LogicField({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border p-4">
      <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">{label}</p>
      <p className="mt-2 text-sm text-slate-900">{value}</p>
    </div>
  );
}

export function RuleLogic({ rule }: { rule: DetectionRule }) {
  const serialized = serializeRuleContent(rule.rule_type, rule.rule_content as SigmaRuleContent);

  if (rule.rule_type === 'sigma') {
    return (
      <div className="space-y-4">
        <div>
          <p className="text-sm font-medium">Sigma YAML</p>
          <p className="text-sm text-muted-foreground">Read-only detection definition rendered through the Sigma Monaco editor.</p>
        </div>
        <RuleSigmaMonaco value={stringifySigmaContent(serialized)} readOnly height={520} />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
        {Object.entries(serialized).map(([key, value]) => (
          <LogicField key={key} label={key.replace(/_/g, ' ')} value={typeof value === 'string' ? value : JSON.stringify(value)} />
        ))}
      </div>

      <div className="rounded-[26px] border border-[color:var(--card-border)] bg-[var(--card-bg)] p-5 shadow-[var(--card-shadow)]">
        <p className="text-sm font-medium">Raw detection payload</p>
        <pre className="mt-4 overflow-x-auto rounded-2xl bg-slate-950/95 p-4 text-xs text-slate-100">
          {JSON.stringify(serialized, null, 2)}
        </pre>
      </div>
    </div>
  );
}
