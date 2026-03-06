'use client';

import { Menu, Search } from 'lucide-react';
import { useSidebar } from '@/hooks/use-sidebar';
import { useIsMobile } from '@/hooks/use-media-query';
import { useCommandPalette } from '@/hooks/use-command-palette';
import { Breadcrumbs } from './breadcrumbs';
import { NotificationDropdown } from './notification-dropdown';
import { UserMenu } from './user-menu';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';

export function Header() {
  const { toggleMobileOpen } = useSidebar();
  const isMobile = useIsMobile();
  const { setOpen } = useCommandPalette();

  const isMac = typeof window !== 'undefined'
    ? navigator.platform.toUpperCase().includes('MAC')
    : true;
  const shortcutLabel = isMac ? '⌘K' : 'Ctrl+K';

  return (
    <TooltipProvider>
      <header className="sticky top-0 z-30 flex h-16 shrink-0 items-center justify-between border-b bg-card px-4 shadow-sm">
        <div className="flex items-center gap-3 min-w-0 flex-1">
          {isMobile && (
            <button
              onClick={toggleMobileOpen}
              aria-label="Open navigation menu"
              className="rounded-md p-2 text-muted-foreground hover:bg-accent hover:text-accent-foreground"
            >
              <Menu className="h-5 w-5" />
            </button>
          )}
          <Breadcrumbs />
        </div>

        <div className="flex items-center gap-1 shrink-0">
          <Tooltip delayDuration={300}>
            <TooltipTrigger asChild>
              <button
                onClick={() => setOpen(true)}
                aria-label={`Search (${shortcutLabel})`}
                className="rounded-md p-2 text-muted-foreground hover:bg-accent hover:text-accent-foreground"
              >
                <Search className="h-4 w-4" />
              </button>
            </TooltipTrigger>
            <TooltipContent>Search ({shortcutLabel})</TooltipContent>
          </Tooltip>

          <NotificationDropdown />
          <UserMenu />
        </div>
      </header>
    </TooltipProvider>
  );
}
