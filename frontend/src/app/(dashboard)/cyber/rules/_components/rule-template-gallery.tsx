'use client';

import { useQuery } from '@tanstack/react-query';
import { CheckCircle, Zap } from 'lucide-react';
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { ScrollArea } from '@/components/ui/scroll-area';
import { SeverityIndicator } from '@/components/shared/severity-indicator';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { apiGet } from '@/lib/api';
import { normalizeRuleTemplate } from '@/lib/cyber-rules';
import { API_ENDPOINTS } from '@/lib/constants';
import type { RuleTemplate } from '@/types/cyber';

const RULE_TYPE_COLORS: Record<string, string> = {
  sigma: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300',
  threshold: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300',
  correlation: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300',
  anomaly: 'bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-300',
};

interface RuleTemplateGalleryProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  activatedTemplateIds: string[];
  onActivate: (template: RuleTemplate) => void;
}

export function RuleTemplateGallery({
  open,
  onOpenChange,
  activatedTemplateIds,
  onActivate,
}: RuleTemplateGalleryProps) {
  const { data: envelope, isLoading, error, refetch } = useQuery({
    queryKey: ['rule-templates'],
    queryFn: () => apiGet<{ data: RuleTemplate[] }>(API_ENDPOINTS.CYBER_RULE_TEMPLATES),
    enabled: open,
  });

  const templates = (envelope?.data ?? []).map(normalizeRuleTemplate);

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="flex flex-col p-0 sm:max-w-2xl">
        <SheetHeader className="border-b px-6 py-4">
          <SheetTitle>Rule Template Gallery</SheetTitle>
        </SheetHeader>
        <ScrollArea className="flex-1">
          <div className="p-6">
            {isLoading ? (
              <LoadingSkeleton variant="card" count={6} />
            ) : error ? (
              <ErrorState message="Failed to load templates" onRetry={() => void refetch()} />
            ) : templates.length === 0 ? (
              <p className="text-center text-sm text-muted-foreground">No templates available</p>
            ) : (
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
                {templates.map((template) => {
                  const isActivated = activatedTemplateIds.includes(template.id);
                  return (
                    <div
                      key={template.id}
                      className="flex flex-col rounded-xl border bg-card p-4 transition-shadow hover:shadow-sm"
                    >
                      <div className="mb-2 flex items-start justify-between gap-2">
                        <div className="min-w-0 flex-1">
                          <p className="text-sm font-semibold leading-tight">{template.name}</p>
                        </div>
                        <SeverityIndicator severity={template.severity} />
                      </div>
                      <div className="mb-2 flex flex-wrap gap-1">
                        <span
                          className={`rounded-full px-2 py-0.5 text-xs font-medium capitalize ${RULE_TYPE_COLORS[template.rule_type] ?? ''}`}
                        >
                          {template.rule_type}
                        </span>
                        {template.mitre_technique_ids.slice(0, 2).map((id) => (
                          <Badge key={id} variant="outline" className="font-mono text-xs">
                            {id}
                          </Badge>
                        ))}
                        {template.mitre_technique_ids.length > 2 && (
                          <span className="text-xs text-muted-foreground">
                            +{template.mitre_technique_ids.length - 2}
                          </span>
                        )}
                      </div>
                      <p className="mb-3 flex-1 text-xs text-muted-foreground line-clamp-3">
                        {template.description}
                      </p>
                      <div className="mt-auto">
                        {isActivated ? (
                          <div className="flex items-center gap-1.5 text-xs text-green-600 dark:text-green-400">
                            <CheckCircle className="h-3.5 w-3.5" />
                            Active ✓
                          </div>
                        ) : (
                          <Button
                            size="sm"
                            variant="outline"
                            className="w-full"
                            onClick={() => onActivate(template)}
                          >
                            <Zap className="mr-1.5 h-3.5 w-3.5" />
                            Activate
                          </Button>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </ScrollArea>
      </SheetContent>
    </Sheet>
  );
}
