'use client';

import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { StatusBadge } from '@/components/shared/status-badge';
import { actionItemStatusConfig } from '@/lib/status-configs';
import { CreateActionItemDialog } from '@/app/(dashboard)/acta/action-items/_components/create-action-item-dialog';
import type { ActaActionItem, ActaCommittee, ActaMeeting } from '@/types/suites';

interface ActionItemsTabProps {
  meeting: ActaMeeting;
  committee: ActaCommittee;
  items: ActaActionItem[];
}

export function ActionItemsTab({ meeting, committee, items }: ActionItemsTabProps) {
  const [open, setOpen] = useState(false);

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <Button onClick={() => setOpen(true)}>Create Action Item</Button>
      </div>
      <div className="space-y-3">
        {items.length === 0 ? (
          <p className="text-sm text-muted-foreground">No action items are currently linked to this meeting.</p>
        ) : (
          items.map((item) => (
            <div key={item.id} className="rounded-xl border bg-card px-4 py-3">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <p className="font-medium">{item.title}</p>
                  <p className="text-xs text-muted-foreground">
                    {item.assignee_name} • due {item.due_date}
                  </p>
                </div>
                <StatusBadge status={item.status} config={actionItemStatusConfig} size="sm" />
              </div>
              <p className="mt-2 text-sm text-muted-foreground">{item.description}</p>
            </div>
          ))
        )}
      </div>

      <CreateActionItemDialog
        open={open}
        onOpenChange={setOpen}
        committees={[committee]}
        meetings={[meeting]}
        preset={{ meeting_id: meeting.id, committee_id: committee.id }}
      />
    </div>
  );
}
