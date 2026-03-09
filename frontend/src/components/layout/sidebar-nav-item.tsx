'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { cn } from '@/lib/utils';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import type { NavItem } from '@/config/navigation';

interface SidebarNavItemProps {
  item: NavItem;
  collapsed: boolean;
  badgeCount?: number;
}

export function SidebarNavItem({ item, collapsed, badgeCount }: SidebarNavItemProps) {
  const pathname = usePathname();
  const currentPath = pathname ?? '';
  const isActive =
    item.href === '/dashboard'
      ? currentPath === '/dashboard'
      : currentPath.startsWith(item.href);

  const Icon = item.icon;
  const showBadge = badgeCount !== undefined && badgeCount > 0;

  const content = (
    <Link
      href={item.href}
      aria-current={isActive ? 'page' : undefined}
      className={cn(
        'group relative flex items-center gap-3 rounded-2xl px-3 py-2.5 text-sm font-medium transition-all duration-200',
        collapsed ? 'justify-center px-2.5' : 'justify-start',
        isActive
          ? '[background:var(--sidebar-active)] text-slate-900 shadow-[0_18px_40px_-28px_rgba(15,23,42,0.65)]'
          : 'text-slate-300 hover:bg-[var(--sidebar-hover)] hover:text-white',
      )}
    >
      <div className="relative flex shrink-0 items-center justify-center">
        <Icon className={cn('h-4 w-4 transition-transform duration-200', !isActive && 'group-hover:scale-110')} />
        {collapsed && showBadge && (
          <span className="absolute -right-1 -top-1 h-2.5 w-2.5 rounded-full bg-rose-400 shadow-[0_0_0_4px_rgba(15,23,42,0.7)]" />
        )}
      </div>
      {!collapsed && (
        <>
          <span className="flex-1 truncate tracking-[-0.01em]">{item.label}</span>
          {showBadge && item.badge && (
            <span
              className={cn(
                'ml-auto flex h-6 min-w-[1.5rem] items-center justify-center rounded-full px-2 text-[11px] font-semibold shadow-sm',
                item.badge.variant === 'destructive' && 'bg-rose-500 text-white',
                item.badge.variant === 'warning' && 'bg-amber-200 text-amber-950',
                item.badge.variant === 'default' && (isActive ? 'bg-slate-900/10 text-slate-900' : 'bg-white/12 text-white'),
              )}
            >
              {badgeCount}
            </span>
          )}
        </>
      )}
    </Link>
  );

  if (collapsed) {
    return (
      <Tooltip delayDuration={300}>
        <TooltipTrigger asChild>{content}</TooltipTrigger>
        <TooltipContent side="right" className="rounded-xl border border-slate-800/10 bg-slate-950 px-3 py-2 text-white shadow-lg">
          <div className="flex items-center gap-2">
            <span>{item.label}</span>
            {showBadge && (
              <span className="rounded-full bg-rose-500 px-1.5 py-0.5 text-xs text-white">
                {badgeCount}
              </span>
            )}
          </div>
        </TooltipContent>
      </Tooltip>
    );
  }

  return content;
}
