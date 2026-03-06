'use client';

import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';

interface CategoryTab {
  key: string;
  label: string;
  count?: number;
}

const TABS: CategoryTab[] = [
  { key: 'all', label: 'All' },
  { key: 'unread', label: 'Unread' },
  { key: 'security', label: 'Security' },
  { key: 'workflow', label: 'Workflow' },
  { key: 'data', label: 'Data' },
  { key: 'system', label: 'System' },
];

interface NotificationCategoryTabsProps {
  activeTab: string;
  onTabChange: (tab: string) => void;
  unreadCount?: number;
}

export function NotificationCategoryTabs({
  activeTab,
  onTabChange,
  unreadCount = 0,
}: NotificationCategoryTabsProps) {
  return (
    <Tabs value={activeTab} onValueChange={onTabChange}>
      <TabsList className="flex h-auto w-full flex-wrap justify-start gap-1 bg-transparent p-0">
        {TABS.map((tab) => {
          const count = tab.key === 'unread' ? unreadCount : tab.count;
          return (
            <TabsTrigger
              key={tab.key}
              value={tab.key}
              className={cn(
                'relative flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm',
                tab.key === 'unread' && (count ?? 0) > 0 && 'text-primary',
              )}
            >
              {tab.label}
              {count !== undefined && count > 0 && (
                <Badge
                  variant={tab.key === 'unread' ? 'default' : 'secondary'}
                  className="h-4 min-w-4 px-1 text-[10px]"
                >
                  {count > 99 ? '99+' : count}
                </Badge>
              )}
            </TabsTrigger>
          );
        })}
      </TabsList>
    </Tabs>
  );
}
