import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { NotificationCard } from './notification-card';
import type { Notification } from '@/types/models';
import { ShieldAlert } from 'lucide-react';
import { getNotificationIcon } from '@/lib/notification-utils';
import { TooltipProvider } from '@/components/ui/tooltip';

const baseNotification: Notification = {
  id: 'notification-1',
  type: 'alert.created',
  title: 'Critical Security Alert',
  body: 'A critical alert has been detected.',
  category: 'security',
  priority: 'critical',
  read: false,
  read_at: null,
  action_url: '/cyber/alerts/1',
  data: null,
  created_at: new Date().toISOString(),
};

describe('NotificationCard', () => {
  it('renders unread notifications with unread label', () => {
    renderCard();

    expect(screen.getByLabelText(/critical security alert \(unread\)/i)).toBeInTheDocument();
  });

  it('renders read notifications without unread label', () => {
    renderCard(
      <NotificationCard
        notification={{ ...baseNotification, read: true, read_at: new Date().toISOString() }}
        onMarkRead={vi.fn()}
        onDelete={vi.fn()}
      />,
    );

    expect(screen.getByLabelText(/critical security alert/i)).toBeInTheDocument();
  });

  it('marks unread notifications as read and navigates when clicked', async () => {
    const user = userEvent.setup();
    const onMarkRead = vi.fn();
    const onNavigate = vi.fn();

    renderCard(
      <NotificationCard
        notification={baseNotification}
        onMarkRead={onMarkRead}
        onDelete={vi.fn()}
        onNavigate={onNavigate}
      />,
    );

    await user.click(screen.getByRole('article'));

    expect(onMarkRead).toHaveBeenCalledWith(baseNotification.id);
    expect(onNavigate).toHaveBeenCalledWith('/cyber/alerts/1');
  });

  it('renders hover action buttons', () => {
    renderCard();

    expect(screen.getByLabelText(/mark as read/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/delete notification/i)).toBeInTheDocument();
  });

  it('maps security critical icon to ShieldAlert', () => {
    expect(getNotificationIcon(baseNotification)).toBe(ShieldAlert);
  });
});

function renderCard(
  card: React.ReactNode = (
    <NotificationCard
      notification={baseNotification}
      onMarkRead={vi.fn()}
      onDelete={vi.fn()}
    />
  ),
) {
  return render(<TooltipProvider>{card}</TooltipProvider>);
}
