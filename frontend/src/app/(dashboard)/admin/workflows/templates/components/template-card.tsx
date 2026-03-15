'use client';

import { GitBranch, Layers, BarChart3 } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { titleCase, truncate } from '@/lib/format';
import type { WorkflowTemplate } from '@/types/models';

interface TemplateCardProps {
  template: WorkflowTemplate;
  onClick: (template: WorkflowTemplate) => void;
}

export function TemplateCard({ template, onClick }: TemplateCardProps) {
  return (
    <Card
      className="cursor-pointer hover:shadow-md transition-shadow"
      onClick={() => onClick(template)}
    >
      <CardHeader className="pb-2">
        <div className="flex items-start justify-between gap-2">
          <CardTitle className="text-sm font-semibold leading-snug line-clamp-2">
            {template.name}
          </CardTitle>
          <Badge variant="secondary" className="text-[10px] shrink-0">
            {titleCase(template.category)}
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        {template.description && (
          <p className="text-xs text-muted-foreground line-clamp-3">
            {truncate(template.description, 120)}
          </p>
        )}
        <div className="flex items-center gap-3 text-xs text-muted-foreground">
          <span className="flex items-center gap-1">
            <GitBranch className="h-3 w-3" />
            {template.steps?.length ?? 0} steps
          </span>
          <span className="flex items-center gap-1">
            <Layers className="h-3 w-3" />
            {template.variables?.length ?? 0} vars
          </span>
          <span className="flex items-center gap-1">
            <BarChart3 className="h-3 w-3" />
            {template.usage_count ?? 0} uses
          </span>
        </div>
        {(template.tags?.length ?? 0) > 0 && (
          <div className="flex flex-wrap gap-1">
            {template.tags!.slice(0, 4).map((tag) => (
              <Badge key={tag} variant="outline" className="text-[10px]">
                {tag}
              </Badge>
            ))}
            {template.tags!.length > 4 && (
              <Badge variant="outline" className="text-[10px]">
                +{template.tags!.length - 4}
              </Badge>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
