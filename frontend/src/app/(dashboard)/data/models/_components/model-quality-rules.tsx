'use client';

import { Badge } from '@/components/ui/badge';
import { type QualityRule } from '@/lib/data-suite';
import { qualitySeverityVisuals } from '@/lib/data-suite/utils';

interface ModelQualityRulesProps {
  rules: QualityRule[];
}

export function ModelQualityRules({
  rules,
}: ModelQualityRulesProps) {
  if (rules.length === 0) {
    return (
      <div className="rounded-lg border border-dashed p-6 text-sm text-muted-foreground">
        No quality rules are attached to this model.
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {rules.map((rule) => {
        const severity = qualitySeverityVisuals[rule.severity];
        return (
          <div key={rule.id} className="rounded-lg border px-4 py-3">
            <div className="flex items-start justify-between gap-3">
              <div>
                <div className="font-medium">{rule.name}</div>
                <div className="mt-1 text-xs text-muted-foreground">
                  {rule.rule_type} {rule.column_name ? `• ${rule.column_name}` : ''}
                </div>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                <Badge variant="outline" className={severity.className}>
                  {severity.label}
                </Badge>
                <Badge variant="outline">{rule.last_status ?? 'never run'}</Badge>
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}
