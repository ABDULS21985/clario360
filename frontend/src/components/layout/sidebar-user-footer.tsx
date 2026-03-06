'use client';

import { useAuth } from '@/hooks/use-auth';
import { cn } from '@/lib/utils';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { LogOut, Settings, User } from 'lucide-react';
import Link from 'next/link';

interface SidebarUserFooterProps {
  collapsed: boolean;
}

function getInitials(firstName: string, lastName: string): string {
  return `${firstName?.charAt(0) ?? ''}${lastName?.charAt(0) ?? ''}`.toUpperCase() || 'U';
}

function getAvatarColor(userId: string): string {
  const colors = [
    'bg-blue-500',
    'bg-purple-500',
    'bg-green-500',
    'bg-orange-500',
    'bg-pink-500',
    'bg-teal-500',
    'bg-indigo-500',
    'bg-red-500',
  ];
  let hash = 0;
  for (let i = 0; i < userId.length; i++) {
    hash = (hash * 31 + userId.charCodeAt(i)) >>> 0;
  }
  return colors[hash % colors.length];
}

export function SidebarUserFooter({ collapsed }: SidebarUserFooterProps) {
  const { user, logout } = useAuth();

  if (!user) return null;

  const initials = getInitials(user.first_name, user.last_name);
  const avatarColor = getAvatarColor(user.id);
  const primaryRole = user.roles?.[0]?.name ?? 'Viewer';
  const fullName = `${user.first_name} ${user.last_name}`.trim() || user.email;

  const avatar = (
    <div
      className={cn(
        'flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-xs font-semibold text-white',
        avatarColor,
      )}
      aria-hidden="true"
    >
      {initials}
    </div>
  );

  if (collapsed) {
    return (
      <div className="flex flex-col items-center gap-1 p-2">
        <Tooltip delayDuration={300}>
          <TooltipTrigger asChild>
            <Link href="/settings">{avatar}</Link>
          </TooltipTrigger>
          <TooltipContent side="right">
            <div>
              <p className="font-medium">{fullName}</p>
              <p className="text-xs text-muted-foreground">{primaryRole}</p>
            </div>
          </TooltipContent>
        </Tooltip>
        <Tooltip delayDuration={300}>
          <TooltipTrigger asChild>
            <button
              onClick={logout}
              className="rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-accent-foreground"
              aria-label="Sign out"
            >
              <LogOut className="h-3.5 w-3.5" />
            </button>
          </TooltipTrigger>
          <TooltipContent side="right">Sign out</TooltipContent>
        </Tooltip>
      </div>
    );
  }

  return (
    <div className="flex items-center gap-2 p-3">
      <Link href="/settings">{avatar}</Link>
      <div className="flex-1 overflow-hidden">
        <p className="truncate text-sm font-medium">{fullName}</p>
        <p className="truncate text-xs text-muted-foreground">{primaryRole}</p>
      </div>
      <div className="flex items-center gap-1">
        <Link
          href="/settings"
          className="rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-accent-foreground"
          aria-label="Settings"
        >
          <Settings className="h-3.5 w-3.5" />
        </Link>
        <button
          onClick={logout}
          className="rounded-md p-1 text-muted-foreground hover:bg-accent hover:text-accent-foreground"
          aria-label="Sign out"
        >
          <LogOut className="h-3.5 w-3.5" />
        </button>
      </div>
    </div>
  );
}
