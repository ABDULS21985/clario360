'use client';

import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';
import type { TaskCounts } from '@/types/models';

interface TaskStatusTabsProps {
  activeTab: string;
  onTabChange: (tab: string) => void;
  counts?: TaskCounts;
}

export function TaskStatusTabs({ activeTab, onTabChange, counts }: TaskStatusTabsProps) {
  const allCount = counts
    ? counts.pending +
      counts.claimed_by_me +
      counts.completed +
      counts.overdue +
      counts.escalated
    : undefined;

  const tabs = [
    { key: 'all', label: 'All', count: allCount },
    { key: 'pending', label: 'Pending', count: counts?.pending },
    { key: 'claimed', label: 'Claimed', count: counts?.claimed_by_me },
    { key: 'completed', label: 'Completed', count: counts?.completed },
    { key: 'overdue', label: 'Overdue', count: counts?.overdue, urgent: true },
  ];

  return (
    <Tabs value={activeTab} onValueChange={onTabChange}>
      <TabsList className="flex h-auto w-full flex-wrap justify-start gap-1 bg-transparent p-0">
        {tabs.map((tab) => (
          <TabsTrigger
            key={tab.key}
            value={tab.key}
            className={cn(
              'flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm',
              tab.urgent && (tab.count ?? 0) > 0 && 'text-destructive',
            )}
          >
            {tab.label}
            {tab.count !== undefined && (
              <Badge
                variant={tab.urgent && tab.count > 0 ? 'destructive' : 'secondary'}
                className="h-4 min-w-4 px-1 text-[10px]"
              >
                {tab.count}
              </Badge>
            )}
          </TabsTrigger>
        ))}
      </TabsList>
    </Tabs>
  );
}
