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
          'fixed inset-y-3 left-3 z-50 flex w-[calc(100vw-1.5rem)] max-w-[320px] flex-col overflow-hidden rounded-[30px] border border-[color:var(--sidebar-border)] [background:var(--sidebar-bg)] text-white shadow-[0_35px_85px_-45px_rgba(15,23,42,0.9)]',
          'animate-in slide-in-from-left-0 duration-200',
        )}
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-white/10 px-4 py-4">
          <Link
            href="/dashboard"
            className="flex items-center gap-3"
            onClick={() => setMobileOpen(false)}
          >
            <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-[linear-gradient(135deg,#d1fae5,#fef3c7)] text-base font-bold tracking-tight text-slate-950 shadow-sm">
              C
            </div>
            <div>
              <p className="text-base font-semibold tracking-[-0.03em] text-white">
                Clario 360
              </p>
              <p className="text-[11px] uppercase tracking-[0.26em] text-slate-300">
                Enterprise Grid
              </p>
            </div>
          </Link>
          <button
            onClick={() => setMobileOpen(false)}
            aria-label="Close navigation menu"
            className="rounded-2xl border border-white/10 bg-white/6 p-2 text-slate-300 transition-colors hover:bg-white/10 hover:text-white"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Navigation */}
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

        <div className="border-t border-white/10 px-3 pb-3 pt-1">
          <SidebarUserFooter collapsed={false} />
        </div>
      </aside>
    </TooltipProvider>
  );
}
