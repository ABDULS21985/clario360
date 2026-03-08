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
        'group relative flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
        collapsed ? 'justify-center px-2' : 'justify-start',
        isActive
          ? 'border-l-2 border-primary bg-primary/10 text-primary'
          : 'text-muted-foreground hover:bg-accent/50 hover:text-accent-foreground border-l-2 border-transparent',
      )}
    >
      <div className="relative flex shrink-0 items-center justify-center">
        <Icon className="h-4 w-4" />
        {collapsed && showBadge && (
          <span className="absolute -right-1 -top-1 h-2 w-2 rounded-full bg-destructive" />
        )}
      </div>
      {!collapsed && (
        <>
          <span className="flex-1 truncate">{item.label}</span>
          {showBadge && item.badge && (
            <span
              className={cn(
                'ml-auto flex h-5 min-w-[1.25rem] items-center justify-center rounded-full px-1.5 text-xs font-semibold',
                item.badge.variant === 'destructive' && 'bg-destructive text-destructive-foreground',
                item.badge.variant === 'warning' && 'bg-amber-100 text-amber-800',
                item.badge.variant === 'default' && 'bg-primary/10 text-primary',
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
        <TooltipContent side="right">
          <div className="flex items-center gap-2">
            <span>{item.label}</span>
            {showBadge && (
              <span className="rounded-full bg-destructive px-1.5 py-0.5 text-xs text-destructive-foreground">
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
