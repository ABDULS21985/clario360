'use client';

import Link from 'next/link';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { type DataModel, type QualityRule } from '@/lib/data-suite';
import { qualitySeverityVisuals } from '@/lib/data-suite/utils';

interface SourceQualityTabProps {
  models: DataModel[];
  rules: QualityRule[];
}

export function SourceQualityTab({
  models,
  rules,
}: SourceQualityTabProps) {
  const modelIds = new Set(models.map((model) => model.id));
  const relevantRules = rules.filter((rule) => modelIds.has(rule.model_id));

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>Models Derived From This Source</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {models.length === 0 ? (
            <p className="text-sm text-muted-foreground">No governed models have been derived from this source yet.</p>
          ) : (
            models.map((model) => (
              <div key={model.id} className="rounded-lg border px-4 py-3">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <Link href={`/data/models/${model.id}`} className="font-medium hover:text-primary">
                      {model.display_name || model.name}
                    </Link>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {model.field_count} fields • {model.contains_pii ? 'Contains PII' : 'No PII'}
                    </p>
                  </div>
                  <Badge variant="outline">{model.data_classification}</Badge>
                </div>
              </div>
            ))
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Quality Rules</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {relevantRules.length === 0 ? (
            <p className="text-sm text-muted-foreground">No quality rules are attached to models from this source.</p>
          ) : (
            relevantRules.map((rule) => {
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
                    <Badge variant="outline" className={severity.className}>
                      {severity.label}
                    </Badge>
                  </div>
                </div>
              );
            })
          )}
        </CardContent>
      </Card>
    </div>
  );
}
