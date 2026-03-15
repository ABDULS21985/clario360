'use client';

import {
  DollarSign,
  Calendar,
  User,
  Shield,
  Lightbulb,
  ArrowDownRight,
  Tag,
  Hash,
} from 'lucide-react';
import { DetailPanel } from '@/components/shared/detail-panel';
import { StatusBadge } from '@/components/shared/status-badge';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { budgetItemStatusConfig } from '@/lib/status-configs';
import { formatDate, formatCurrency, titleCase } from '@/lib/format';
import type { VCISOBudgetItem } from '@/types/cyber';

interface BudgetDetailPanelProps {
  item: VCISOBudgetItem | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function BudgetDetailPanel({
  item,
  open,
  onOpenChange,
}: BudgetDetailPanelProps) {
  if (!item) return null;

  return (
    <DetailPanel
      open={open}
      onOpenChange={onOpenChange}
      title={item.title}
      description="Budget item details and justification"
      width="lg"
    >
      <div className="space-y-6">
        {/* Status, Type, Category */}
        <div className="flex flex-wrap items-center gap-2">
          <StatusBadge status={item.status} config={budgetItemStatusConfig} />
          <Badge
            variant="secondary"
            className={
              item.type === 'capex'
                ? 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300'
                : 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300'
            }
          >
            {item.type === 'capex' ? 'CapEx' : 'OpEx'}
          </Badge>
          <Badge variant="outline" className="text-xs">
            <Tag className="mr-1 h-3 w-3" />
            {item.category}
          </Badge>
        </div>

        {/* Financial Summary */}
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div className="rounded-lg border p-3">
            <p className="text-xs text-muted-foreground flex items-center gap-1">
              <DollarSign className="h-3 w-3" />
              Amount
            </p>
            <p className="text-lg font-bold mt-0.5">
              {formatCurrency(item.amount, item.currency)}
            </p>
          </div>
          <div className="rounded-lg border p-3">
            <p className="text-xs text-muted-foreground flex items-center gap-1">
              <ArrowDownRight className="h-3 w-3" />
              Risk Reduction
            </p>
            <p className="text-lg font-bold mt-0.5 text-green-600">
              {item.risk_reduction_estimate}%
            </p>
          </div>
          <div className="rounded-lg border p-3">
            <p className="text-xs text-muted-foreground flex items-center gap-1">
              <Hash className="h-3 w-3" />
              Priority
            </p>
            <p className="text-lg font-bold mt-0.5">
              {item.priority}
            </p>
          </div>
          <div className="rounded-lg border p-3">
            <p className="text-xs text-muted-foreground flex items-center gap-1">
              <Calendar className="h-3 w-3" />
              Fiscal Year
            </p>
            <p className="text-lg font-bold mt-0.5">
              {item.fiscal_year}
              {item.quarter ? ` ${item.quarter}` : ''}
            </p>
          </div>
        </div>

        <Separator />

        {/* Justification */}
        <div>
          <h4 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-2">
            Business Justification
          </h4>
          <p className="text-sm leading-relaxed text-foreground whitespace-pre-wrap">
            {item.justification}
          </p>
        </div>

        <Separator />

        {/* Linked Risks */}
        <div>
          <h4 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-2">
            <Shield className="inline h-3.5 w-3.5 mr-1 -mt-0.5" />
            Linked Risks ({item.linked_risk_ids.length})
          </h4>
          <div className="flex flex-wrap gap-1.5">
            {item.linked_risk_ids.length > 0 ? (
              item.linked_risk_ids.map((id) => (
                <Badge key={id} variant="secondary" className="text-xs font-mono">
                  {id}
                </Badge>
              ))
            ) : (
              <span className="text-sm text-muted-foreground">
                No risks linked
              </span>
            )}
          </div>
        </div>

        {/* Linked Recommendations */}
        <div>
          <h4 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground mb-2">
            <Lightbulb className="inline h-3.5 w-3.5 mr-1 -mt-0.5" />
            Linked Recommendations ({item.linked_recommendation_ids.length})
          </h4>
          <div className="flex flex-wrap gap-1.5">
            {item.linked_recommendation_ids.length > 0 ? (
              item.linked_recommendation_ids.map((id) => (
                <Badge key={id} variant="secondary" className="text-xs font-mono">
                  {id}
                </Badge>
              ))
            ) : (
              <span className="text-sm text-muted-foreground">
                No recommendations linked
              </span>
            )}
          </div>
        </div>

        <Separator />

        {/* Owner and Dates */}
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          {item.owner_name && (
            <div className="rounded-lg border p-3">
              <p className="text-xs text-muted-foreground flex items-center gap-1">
                <User className="h-3 w-3" />
                Owner
              </p>
              <p className="text-sm font-medium mt-0.5">{item.owner_name}</p>
            </div>
          )}
          <div className="rounded-lg border p-3">
            <p className="text-xs text-muted-foreground flex items-center gap-1">
              <Calendar className="h-3 w-3" />
              Created
            </p>
            <p className="text-sm font-medium mt-0.5">
              {formatDate(item.created_at)}
            </p>
          </div>
          <div className="rounded-lg border p-3">
            <p className="text-xs text-muted-foreground flex items-center gap-1">
              <Calendar className="h-3 w-3" />
              Updated
            </p>
            <p className="text-sm font-medium mt-0.5">
              {formatDate(item.updated_at)}
            </p>
          </div>
        </div>
      </div>
    </DetailPanel>
  );
}
