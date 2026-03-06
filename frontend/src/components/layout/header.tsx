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
  const { user, logout } = useAuth();

  return (
    <header className="flex h-14 items-center justify-between border-b bg-card px-6">
      <div className="flex items-center gap-4">
        <h2 className="text-sm font-medium text-muted-foreground">
          Enterprise AI Platform
        </h2>
      </div>

      <div className="flex items-center gap-4">
        {/* Notifications */}
        <button
          className="rounded-md p-2 text-muted-foreground hover:bg-accent hover:text-accent-foreground"
          aria-label="Notifications"
        >
          <span className="text-sm">Bell</span>
        </button>

        {/* User menu */}
        <div className="flex items-center gap-2">
          <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary text-xs font-medium text-primary-foreground">
            {user?.email?.charAt(0).toUpperCase() || "U"}
          </div>
          <div className="hidden md:block">
            <p className="text-sm font-medium">{user?.email || "User"}</p>
            <p className="text-xs text-muted-foreground">
              {user?.roles?.[0]?.name ?? 'viewer'}
            </p>
          </div>
          <button
            onClick={logout}
            className="ml-2 rounded-md px-3 py-1.5 text-xs font-medium text-muted-foreground hover:bg-accent hover:text-accent-foreground"
          >
            Sign out
          </button>
        </div>
      </div>
    </header>
  );
}
