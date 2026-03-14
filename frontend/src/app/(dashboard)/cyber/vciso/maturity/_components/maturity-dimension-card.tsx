'use client';

import { useState } from 'react';
import {
  ChevronDown,
  ChevronUp,
  AlertTriangle,
  Lightbulb,
  Users,
  Cog,
  Cpu,
} from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { Separator } from '@/components/ui/separator';
import { cn } from '@/lib/utils';
import type { VCISOMaturityDimension, MaturityCategory } from '@/types/cyber';

interface MaturityDimensionCardProps {
  dimension: VCISOMaturityDimension;
  onViewDetails?: (dimension: VCISOMaturityDimension) => void;
}

const CATEGORY_CONFIG: Record<
  MaturityCategory,
  { label: string; color: string; bgColor: string; icon: typeof Users }
> = {
  people: {
    label: 'People',
    color: 'text-blue-700 dark:text-blue-400',
    bgColor: 'bg-blue-100 dark:bg-blue-900/30',
    icon: Users,
  },
  process: {
    label: 'Process',
    color: 'text-green-700 dark:text-green-400',
    bgColor: 'bg-green-100 dark:bg-green-900/30',
    icon: Cog,
  },
  technology: {
    label: 'Technology',
    color: 'text-purple-700 dark:text-purple-400',
    bgColor: 'bg-purple-100 dark:bg-purple-900/30',
    icon: Cpu,
  },
  governance: {
    label: 'Governance',
    color: 'text-indigo-700 dark:text-indigo-400',
    bgColor: 'bg-indigo-100 dark:bg-indigo-900/30',
    icon: Cog,
  },
  security: {
    label: 'Security',
    color: 'text-red-700 dark:text-red-400',
    bgColor: 'bg-red-100 dark:bg-red-900/30',
    icon: Cpu,
  },
  operations: {
    label: 'Operations',
    color: 'text-amber-700 dark:text-amber-400',
    bgColor: 'bg-amber-100 dark:bg-amber-900/30',
    icon: Users,
  },
};

function getLevelColor(level: number): string {
  if (level >= 4) return 'text-green-600';
  if (level >= 3) return 'text-blue-600';
  if (level >= 2) return 'text-amber-600';
  return 'text-red-600';
}

function getProgressColor(current: number, target: number): string {
  const ratio = target > 0 ? current / target : 0;
  if (ratio >= 0.8) return 'bg-green-500';
  if (ratio >= 0.5) return 'bg-amber-500';
  return 'bg-red-500';
}

export function MaturityDimensionCard({
  dimension,
  onViewDetails,
}: MaturityDimensionCardProps) {
  const [expanded, setExpanded] = useState(false);
  const categoryConfig = CATEGORY_CONFIG[dimension.category];
  const CategoryIcon = categoryConfig.icon;
  const progressPct =
    dimension.target_level > 0
      ? Math.min((dimension.current_level / dimension.target_level) * 100, 100)
      : 0;

  return (
    <Card className="group overflow-hidden transition-shadow hover:shadow-md">
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between gap-2">
          <div className="space-y-1.5 min-w-0">
            <CardTitle className="text-sm font-semibold leading-tight truncate">
              {dimension.name}
            </CardTitle>
            <Badge
              variant="secondary"
              className={cn(
                'text-xs font-medium',
                categoryConfig.bgColor,
                categoryConfig.color,
              )}
            >
              <CategoryIcon className="mr-1 h-3 w-3" />
              {categoryConfig.label}
            </Badge>
          </div>
          <div className="text-right shrink-0">
            <p
              className={cn(
                'text-2xl font-bold tabular-nums',
                getLevelColor(dimension.current_level),
              )}
            >
              {dimension.score.toFixed(1)}
            </p>
            <p className="text-xs text-muted-foreground">Score</p>
          </div>
        </div>
      </CardHeader>

      <CardContent className="space-y-3">
        {/* Level Progress */}
        <div>
          <div className="flex items-center justify-between text-sm mb-1.5">
            <span className="text-muted-foreground">
              Level {dimension.current_level} / {dimension.target_level}
            </span>
            <span className="text-xs font-medium">
              {progressPct.toFixed(0)}%
            </span>
          </div>
          <div className="h-2 w-full rounded-full bg-muted overflow-hidden">
            <div
              className={cn(
                'h-full rounded-full transition-all duration-500',
                getProgressColor(dimension.current_level, dimension.target_level),
              )}
              style={{ width: `${progressPct}%` }}
            />
          </div>
        </div>

        {/* Findings & Recommendations counts */}
        <div className="grid grid-cols-2 gap-2">
          <div className="flex items-center gap-1.5 rounded-lg border p-2">
            <AlertTriangle className="h-3.5 w-3.5 text-amber-500 shrink-0" />
            <div>
              <p className="text-xs text-muted-foreground">Findings</p>
              <p className="text-sm font-semibold tabular-nums">
                {dimension.findings.length}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-1.5 rounded-lg border p-2">
            <Lightbulb className="h-3.5 w-3.5 text-blue-500 shrink-0" />
            <div>
              <p className="text-xs text-muted-foreground">Recommendations</p>
              <p className="text-sm font-semibold tabular-nums">
                {dimension.recommendations.length}
              </p>
            </div>
          </div>
        </div>

        {/* Expand/Collapse */}
        <Button
          variant="ghost"
          size="sm"
          className="w-full text-xs"
          onClick={() => setExpanded(!expanded)}
        >
          {expanded ? (
            <>
              <ChevronUp className="mr-1 h-3.5 w-3.5" />
              Hide Details
            </>
          ) : (
            <>
              <ChevronDown className="mr-1 h-3.5 w-3.5" />
              Show Details
            </>
          )}
        </Button>

        {expanded && (
          <div className="space-y-3 pt-1">
            {/* Findings List */}
            {dimension.findings.length > 0 && (
              <div>
                <h5 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-2 flex items-center gap-1">
                  <AlertTriangle className="h-3 w-3 text-amber-500" />
                  Findings
                </h5>
                <ul className="space-y-1.5">
                  {dimension.findings.map((finding, idx) => (
                    <li
                      key={idx}
                      className="text-sm text-foreground rounded-lg border border-amber-200/60 bg-amber-50/50 dark:bg-amber-900/10 dark:border-amber-800/30 px-3 py-2"
                    >
                      {finding}
                    </li>
                  ))}
                </ul>
              </div>
            )}

            {dimension.findings.length > 0 &&
              dimension.recommendations.length > 0 && <Separator />}

            {/* Recommendations List */}
            {dimension.recommendations.length > 0 && (
              <div>
                <h5 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-2 flex items-center gap-1">
                  <Lightbulb className="h-3 w-3 text-blue-500" />
                  Recommendations
                </h5>
                <ul className="space-y-1.5">
                  {dimension.recommendations.map((rec, idx) => (
                    <li
                      key={idx}
                      className="text-sm text-foreground rounded-lg border border-blue-200/60 bg-blue-50/50 dark:bg-blue-900/10 dark:border-blue-800/30 px-3 py-2"
                    >
                      {rec}
                    </li>
                  ))}
                </ul>
              </div>
            )}

            {/* View Full Details button */}
            {onViewDetails && (
              <>
                <Separator />
                <Button
                  variant="outline"
                  size="sm"
                  className="w-full"
                  onClick={() => onViewDetails(dimension)}
                >
                  View Full Details
                </Button>
              </>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
