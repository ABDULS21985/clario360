import { toast } from 'sonner';
import type { Notification } from '@/types/models';
import { getNotificationIcon } from '@/lib/notification-utils';
import { truncate } from '@/lib/utils';

export function showSuccess(title: string, description?: string): void {
  toast.success(title, { description, duration: 4000 });
}

export function showError(title: string, description?: string): void {
  toast.error(title, { description, duration: 6000 });
}

export function showWarning(title: string, description?: string): void {
  toast.warning(title, { description, duration: 5000 });
}

export function showInfo(title: string, description?: string): void {
  toast.info(title, { description, duration: 4000 });
}

export function showNotificationToast(notification: Notification): void {
  const Icon = getNotificationIcon(notification);
  toast(notification.title, {
    description: truncate(notification.body, 100),
    duration: 6000,
    action: notification.action_url
      ? {
          label: 'View',
          onClick: () => {
            if (typeof window !== 'undefined') {
              window.location.href = notification.action_url!;
            }
          },
        }
      : undefined,
  });
  // Suppress icon warning — toast doesn't accept React elements directly
  void Icon;
}

export function showApiError(error: unknown): void {
  let message = 'An unexpected error occurred.';
  if (error && typeof error === 'object' && 'message' in error) {
    message = (error as { message: string }).message;
  }
  toast.error('Error', { description: message, duration: 6000 });
}
