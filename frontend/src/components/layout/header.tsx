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
      <header className="sticky top-0 z-30 flex h-[78px] shrink-0 items-center justify-between border-b border-border/60 bg-[linear-gradient(180deg,rgba(255,255,255,0.92),rgba(255,255,255,0.78))] px-4 shadow-[0_18px_36px_-36px_rgba(15,23,42,0.55)] backdrop-blur-xl sm:px-5 lg:px-6">
        <div className="flex min-w-0 flex-1 items-center gap-3">
          {isMobile && (
            <button
              onClick={toggleMobileOpen}
              aria-label="Open navigation menu"
              className="inline-flex h-11 w-11 items-center justify-center rounded-2xl border border-border/70 bg-white/80 text-muted-foreground shadow-sm transition-all hover:border-primary/20 hover:bg-white hover:text-foreground"
            >
              <Menu className="h-5 w-5" />
            </button>
          )}
          <div className="min-w-0">
            <div className="mb-1 hidden items-center gap-2 xl:flex">
              <span className="inline-flex items-center gap-2 rounded-full border border-emerald-200 bg-emerald-50 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.24em] text-emerald-900">
                <span className="h-2 w-2 rounded-full bg-emerald-500" aria-hidden="true" />
                Enterprise Console
              </span>
            </div>
            <Breadcrumbs />
          </div>
        </div>

        <div className="flex shrink-0 items-center gap-2">
          <Tooltip delayDuration={300}>
            <TooltipTrigger asChild>
              <button
                onClick={() => setOpen(true)}
                aria-label={`Search (${shortcutLabel})`}
                className="inline-flex h-11 items-center gap-3 rounded-2xl border border-border/70 bg-white/80 px-3 text-sm text-muted-foreground shadow-sm transition-all hover:border-primary/25 hover:bg-white hover:text-foreground"
              >
                <Search className="h-4 w-4 text-primary" />
                {!isMobile && (
                  <>
                    <span className="hidden lg:inline text-sm font-medium text-foreground/80">
                      Search, jump, or run
                    </span>
                    <span className="rounded-xl bg-slate-900 px-2 py-1 text-[11px] font-semibold tracking-wide text-white">
                      {shortcutLabel}
                    </span>
                  </>
                )}
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
