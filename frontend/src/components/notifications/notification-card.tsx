'use client';

import { useRouter } from 'next/navigation';
import { cn } from '@/lib/utils';
import { getNotificationIcon, getNotificationIconColor } from '@/lib/notification-utils';
import { NotificationActions } from './notification-actions';
import { RelativeTime } from '@/components/shared/relative-time';
import { isNotificationRead } from '@/lib/notification-utils';
import type { Notification } from '@/types/models';

interface NotificationCardProps {
  notification: Notification;
  onMarkRead: (id: string) => void;
  onDelete: (id: string) => void;
  onNavigate?: (url: string) => void;
  isNew?: boolean;
}

export function NotificationCard({
  notification,
  onMarkRead,
  onDelete,
  onNavigate,
  isNew = false,
}: NotificationCardProps) {
  const router = useRouter();
  const Icon = getNotificationIcon(notification);
  const iconColor = getNotificationIconColor(notification);
  const isUnread = !isNotificationRead(notification);

  const handleCardClick = () => {
    if (isUnread) {
      onMarkRead(notification.id);
    }
    if (notification.action_url) {
      if (onNavigate) {
        onNavigate(notification.action_url);
      } else {
        router.push(notification.action_url);
      }
    }
  };

  return (
    <article
      role="article"
      aria-label={`${notification.title}${isUnread ? ' (unread)' : ''}`}
      onClick={handleCardClick}
      className={cn(
        'group relative flex items-start gap-3 px-4 py-3 transition-colors',
        (notification.action_url || isUnread) && 'cursor-pointer hover:bg-muted/50',
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
        <div className="mt-1 text-xs text-muted-foreground">
          <RelativeTime date={notification.created_at} />
        </div>
      </div>

      <NotificationActions
        isUnread={isUnread}
        onMarkRead={() => onMarkRead(notification.id)}
        onDelete={() => onDelete(notification.id)}
      />
    </article>
  );
}
