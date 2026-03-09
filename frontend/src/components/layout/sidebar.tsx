'use client';

import { ChevronLeft, ChevronRight } from 'lucide-react';
import Link from 'next/link';
import { cn } from '@/lib/utils';
import { useSidebar } from '@/hooks/use-sidebar';
import { useAuth } from '@/hooks/use-auth';
import { useBadgeCounts } from '@/hooks/use-badge-counts';
import { useIsMobile } from '@/hooks/use-media-query';
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

export function Sidebar() {
  const { collapsed, toggleCollapsed } = useSidebar();
  const { hasPermission } = useAuth();
  const isMobile = useIsMobile();
  const badgeCounts = useBadgeCounts(ALL_BADGE_CONFIGS);

  if (isMobile) return null;

  return (
    <TooltipProvider>
      <aside
        aria-label="Main navigation"
        className={cn(
          'flex h-full shrink-0 flex-col overflow-hidden rounded-[30px] border border-[color:var(--sidebar-border)] bg-[var(--sidebar-bg)] text-white shadow-[0_35px_85px_-45px_rgba(15,23,42,0.9)] transition-[width] duration-300 ease-out',
          collapsed ? 'w-[88px]' : 'w-[292px]',
        )}
      >
        <div className="border-b border-white/10 px-3 py-4">
          {!collapsed ? (
            <Link
              href="/dashboard"
              className="flex items-center gap-3 rounded-[24px] border border-white/10 bg-white/5 px-3 py-3 shadow-[inset_0_1px_0_rgba(255,255,255,0.08)] transition-colors hover:bg-white/8"
            >
              <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-[linear-gradient(135deg,#d1fae5,#fef3c7)] text-base font-bold tracking-tight text-slate-950 shadow-sm">
                C
              </div>
              <div className="min-w-0">
                <p className="truncate text-base font-semibold tracking-[-0.03em] text-white">
                  Clario 360
                </p>
                <p className="truncate text-[11px] uppercase tracking-[0.28em] text-slate-300">
                  Enterprise Grid
                </p>
              </div>
            </Link>
          ) : (
            <Link
              href="/dashboard"
              className="mx-auto flex h-12 w-12 items-center justify-center rounded-2xl bg-[linear-gradient(135deg,#d1fae5,#fef3c7)] text-sm font-bold tracking-tight text-slate-950 shadow-sm"
            >
              C
            </Link>
          )}
        </div>

        <nav className="sidebar-scroll flex-1 overflow-y-auto px-3 py-4">
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
              <SidebarSection key={section.id} label={section.label} collapsed={collapsed}>
                {visibleItems.map((item) => {
                  const count = item.badge ? badgeCounts.get(item.badge.endpoint) : undefined;
                  return (
                    <SidebarNavItem
                      key={item.id}
                      item={item}
                      collapsed={collapsed}
                      badgeCount={count}
                    />
                  );
                })}
              </SidebarSection>
            );
          })}
        </nav>

        <div className="border-t border-white/10 px-3 py-3">
          <button
            onClick={toggleCollapsed}
            aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
            className="flex w-full items-center justify-center rounded-2xl border border-white/10 bg-white/5 p-2 text-slate-300 transition-all hover:bg-white/10 hover:text-white"
          >
            {collapsed ? (
              <ChevronRight className="h-4 w-4" />
            ) : (
              <div className="flex w-full items-center justify-between px-1 text-xs font-medium uppercase tracking-[0.18em]">
                <span>Collapse</span>
                <ChevronLeft className="h-4 w-4" />
              </div>
            )}
          </button>
        </div>

        <div className="border-t border-white/10 px-3 pb-3 pt-1">
          <SidebarUserFooter collapsed={collapsed} />
        </div>
      </aside>
    </TooltipProvider>
  );
}
