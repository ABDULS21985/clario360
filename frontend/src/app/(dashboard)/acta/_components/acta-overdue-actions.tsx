'use client';

import Link from 'next/link';
import { differenceInCalendarDays, parseISO } from 'date-fns';
import { ArrowRight, ClipboardList } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { SectionCard } from '@/components/suites/section-card';
import { EmptyState } from '@/components/common/empty-state';
import { Badge } from '@/components/ui/badge';
import type { ActaActionItemSummary } from '@/types/suites';

interface ActaOverdueActionsProps {
  items: ActaActionItemSummary[];
}

export function ActaOverdueActions({ items }: ActaOverdueActionsProps) {
  return (
    <SectionCard
      title="Overdue Action Items"
      description="Top overdue follow-ups across committees."
      actions={
        <Button variant="ghost" size="sm" asChild>
          <Link href="/acta/action-items?status=overdue">
            Open tracker
            <ArrowRight className="ml-1 h-3.5 w-3.5" />
          </Link>
        </Button>
      }
    >
      {items.length === 0 ? (
        <EmptyState
          icon={ClipboardList}
          title="No overdue items"
          description="Action items are currently within due dates."
        />
      ) : (
        <div className="space-y-3">
          {items.slice(0, 10).map((item) => {
            const daysOverdue = Math.max(
              differenceInCalendarDays(new Date(), parseISO(item.due_date)),
              0,
            );
            return (
              <Link
                key={item.id}
                href={`/acta/action-items?status=overdue`}
                className="block rounded-xl border px-4 py-3 transition hover:border-primary"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <p className="truncate font-medium">{item.title}</p>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {item.committee_name} • {item.assignee_name}
                    </p>
                  </div>
                  <Badge variant="destructive" className="capitalize">
                    {item.priority}
                  </Badge>
                </div>
                <div className="mt-3 flex flex-wrap gap-3 text-xs text-muted-foreground">
                  <span className="font-medium text-destructive">
                    {daysOverdue} day{daysOverdue === 1 ? '' : 's'} overdue
                  </span>
                  <span>Due {formatDate(item.due_date)}</span>
                </div>
              </Link>
            );
          })}
        </div>
      )}
    </SectionCard>
  );
}

function formatDate(value: string) {
  return new Date(value).toLocaleDateString();
}
