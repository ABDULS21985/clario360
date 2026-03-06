'use client';

import { X } from 'lucide-react';
import Link from 'next/link';
import { cn } from '@/lib/utils';
import { useSidebar } from '@/hooks/use-sidebar';
import { useAuth } from '@/hooks/use-auth';
import { useBadgeCounts } from '@/hooks/use-badge-counts';
import { TooltipProvider } from '@/components/ui/tooltip';
import { SidebarSection } from './sidebar-section';
import { SidebarNavItem } from './sidebar-nav-item';
import { SidebarUserFooter } from './sidebar-user-footer';
import { navigation, type BadgeConfig } from '@/config/navigation';

function collectBadgeConfigs(): BadgeConfig[] {
  const configs: BadgeConfig[] = [];
  for (const section of navigation) {
    for (const item of section.items) {
      if (item.badge) configs.push(item.badge);
    }
  }
  return configs;
}

const ALL_BADGE_CONFIGS = collectBadgeConfigs();

export function MobileSidebar() {
  const { mobileOpen, setMobileOpen } = useSidebar();
  const { hasPermission } = useAuth();
  const badgeCounts = useBadgeCounts(ALL_BADGE_CONFIGS);

  if (!mobileOpen) return null;

  return (
    <TooltipProvider>
      {/* Backdrop */}
      <div
        className="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm"
        onClick={() => setMobileOpen(false)}
        aria-hidden="true"
      />

      {/* Sidebar panel */}
      <aside
        aria-label="Mobile navigation"
        className={cn(
          'fixed inset-y-0 left-0 z-50 flex w-72 flex-col border-r bg-card shadow-xl',
          'animate-in slide-in-from-left-0 duration-200',
        )}
      >
        {/* Header */}
        <div className="flex h-16 items-center justify-between border-b px-4">
          <Link
            href="/dashboard"
            className="font-bold text-lg text-primary"
            onClick={() => setMobileOpen(false)}
          >
            Clario 360
          </Link>
          <button
            onClick={() => setMobileOpen(false)}
            aria-label="Close navigation menu"
            className="rounded-md p-1.5 text-muted-foreground hover:bg-accent"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Navigation */}
        <nav className="flex-1 overflow-y-auto py-3 px-2">
          {navigation.map((section) => {
            if (section.permission !== '*:read' && !hasPermission(section.permission)) {
              return null;
            }

            const visibleItems = section.items.filter(
              (item) =>
                !item.permission ||
                item.permission === '*:read' ||
                hasPermission(item.permission),
            );

            if (visibleItems.length === 0) return null;

            return (
              <SidebarSection key={section.id} label={section.label} collapsed={false}>
                {visibleItems.map((item) => {
                  const count = item.badge ? badgeCounts.get(item.badge.endpoint) : undefined;
                  return (
                    <div key={item.id} onClick={() => setMobileOpen(false)}>
                      <SidebarNavItem item={item} collapsed={false} badgeCount={count} />
                    </div>
                  );
                })}
              </SidebarSection>
            );
          })}
        </nav>

        <div className="border-t">
          <SidebarUserFooter collapsed={false} />
        </div>
      </aside>
    </TooltipProvider>
  );
}
