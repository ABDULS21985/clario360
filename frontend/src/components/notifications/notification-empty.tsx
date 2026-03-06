import { Bell, CheckCircle, Shield, Workflow, Database, Settings } from 'lucide-react';
import type { LucideIcon } from 'lucide-react';

interface NotificationEmptyProps {
  category?: string;
}

const EMPTY_STATES: Record<string, { icon: LucideIcon; message: string }> = {
  all: { icon: Bell, message: "You're all caught up!" },
  unread: { icon: CheckCircle, message: 'No unread notifications.' },
  security: { icon: Shield, message: 'No security notifications.' },
  workflow: { icon: Workflow, message: 'No workflow notifications.' },
  data: { icon: Database, message: 'No data notifications.' },
  system: { icon: Settings, message: 'No system notifications.' },
  governance: { icon: Bell, message: 'No governance notifications.' },
  legal: { icon: Bell, message: 'No legal notifications.' },
};

export function NotificationEmpty({ category = 'all' }: NotificationEmptyProps) {
  const state = EMPTY_STATES[category] ?? EMPTY_STATES.all!;
  const Icon = state.icon;

  return (
    <div className="flex flex-col items-center justify-center gap-3 py-16 text-center">
      <div className="flex h-14 w-14 items-center justify-center rounded-full bg-muted">
        <Icon className="h-7 w-7 text-muted-foreground" />
      </div>
      <p className="text-sm text-muted-foreground">{state.message}</p>
    </div>
  );
}
