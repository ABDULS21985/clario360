'use client';

import { useRouter } from 'next/navigation';
import { Eye, Trash2 } from 'lucide-react';
import { cn } from '@/lib/utils';
import { getNotificationIcon, getNotificationIconColor } from '@/lib/notification-utils';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import type { Notification } from '@/types/models';

interface NotificationCardProps {
  notification: Notification;
  onMarkRead: (id: string) => void;
  onDelete: (id: string) => void;
  isNew?: boolean;
}

export function NotificationCard({
  notification,
  onMarkRead,
  onDelete,
  isNew = false,
}: NotificationCardProps) {
  const router = useRouter();
  const Icon = getNotificationIcon(notification);
  const iconColor = getNotificationIconColor(notification);
  const isUnread = !notification.read;

  const handleCardClick = () => {
    if (isUnread) {
      onMarkRead(notification.id);
    }
    if (notification.action_url) {
      router.push(notification.action_url);
    }
  };

  const handleMarkRead = (e: React.MouseEvent) => {
    e.stopPropagation();
    onMarkRead(notification.id);
  };

  const handleDelete = (e: React.MouseEvent) => {
    e.stopPropagation();
    onDelete(notification.id);
  };

  // Format relative time
  const timeAgo = (dateStr: string): string => {
    const diff = Date.now() - new Date(dateStr).getTime();
    const mins = Math.floor(diff / 60000);
    if (mins < 1) return 'just now';
    if (mins < 60) return `${mins}m ago`;
    const hrs = Math.floor(mins / 60);
    if (hrs < 24) return `${hrs}h ago`;
    return `${Math.floor(hrs / 24)}d ago`;
  };

  return (
    <article
      role="article"
      aria-label={`${notification.title}${isUnread ? ' (unread)' : ''}`}
      onClick={notification.action_url ? handleCardClick : undefined}
      className={cn(
        'group relative flex items-start gap-3 px-4 py-3 transition-colors',
        notification.action_url && 'cursor-pointer hover:bg-muted/50',
        isUnread && 'bg-primary/5',
        isNew && 'animate-in slide-in-from-top-2 duration-300',
      )}
    >
      {/* Unread indicator */}
      <div className="mt-1.5 shrink-0">
        {isUnread ? (
          <span className="block h-2 w-2 rounded-full bg-primary" aria-hidden />
        ) : (
          <span className="block h-2 w-2" aria-hidden />
        )}
      </div>

      {/* Icon */}
      <div className="mt-0.5 shrink-0">
        <Icon className={cn('h-4 w-4', iconColor)} aria-hidden />
      </div>

      {/* Content */}
      <div className="flex-1 min-w-0">
        <p className={cn('text-sm truncate', isUnread && 'font-medium')}>
          {notification.title}
        </p>
        <p className="mt-0.5 text-xs text-muted-foreground line-clamp-2">
          {notification.body}
        </p>
        <time
          dateTime={notification.created_at}
          className="mt-1 block text-xs text-muted-foreground"
        >
          {timeAgo(notification.created_at)}
        </time>
      </div>

      {/* Hover actions */}
      <div className="flex shrink-0 items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
        {isUnread && (
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                onClick={handleMarkRead}
                aria-label="Mark as read"
                className="rounded p-1 hover:bg-muted"
              >
                <Eye className="h-3.5 w-3.5 text-muted-foreground" />
              </button>
            </TooltipTrigger>
            <TooltipContent><p className="text-xs">Mark as read</p></TooltipContent>
          </Tooltip>
        )}
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              onClick={handleDelete}
              aria-label="Delete notification"
              className="rounded p-1 hover:bg-muted"
            >
              <Trash2 className="h-3.5 w-3.5 text-muted-foreground" />
            </button>
          </TooltipTrigger>
          <TooltipContent><p className="text-xs">Delete notification</p></TooltipContent>
        </Tooltip>
      </div>
    </article>
  );
}
