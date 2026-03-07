'use client';

import { useMemo, useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { CreateActionItemDialog } from '@/app/(dashboard)/acta/action-items/_components/create-action-item-dialog';
import { showApiError, showSuccess } from '@/lib/toast';
import type { ActaActionItem, ActaCommittee, ActaExtractedAction, ActaMeeting } from '@/types/suites';
import { enterpriseApi } from '@/lib/enterprise';

interface AiActionsSidebarProps {
  meeting: ActaMeeting;
  committee: ActaCommittee;
  extracted: ActaExtractedAction[];
  existingItems: ActaActionItem[];
}

export function AiActionsSidebar({
  meeting,
  committee,
  extracted,
  existingItems,
}: AiActionsSidebarProps) {
  const queryClient = useQueryClient();
  const [selected, setSelected] = useState<ActaExtractedAction | null>(null);

  const existingTitles = useMemo(
    () => new Set(existingItems.map((item) => item.title.trim().toLowerCase())),
    [existingItems],
  );

  const createAllMutation = useMutation({
    mutationFn: async () => {
      for (const action of extracted) {
        if (existingTitles.has(action.title.trim().toLowerCase())) {
          continue;
        }
        const member = committee.members?.find((candidate) =>
          candidate.user_name.toLowerCase().includes(action.assigned_to.toLowerCase()),
        );
        if (!member) {
          continue;
        }
        await enterpriseApi.acta.createActionItem({
          meeting_id: meeting.id,
          committee_id: committee.id,
          agenda_item_id: null,
          title: action.title,
          description: action.description,
          priority: action.priority,
          assigned_to: member.user_id,
          assignee_name: member.user_name,
          due_date: action.due_date ?? new Date().toISOString().slice(0, 10),
          tags: ['ai-generated'],
          metadata: { source: action.source },
        });
      }
    },
    onSuccess: async () => {
      showSuccess('Action items created.', 'AI-extracted action items have been pushed to the tracker.');
      await queryClient.invalidateQueries({ queryKey: ['acta-meeting-actions', meeting.id] });
    },
    onError: showApiError,
  });

  return (
    <div className="space-y-3 rounded-xl border bg-card p-4">
      <div className="flex items-center justify-between">
        <div>
          <p className="font-medium">AI-extracted actions</p>
          <p className="text-xs text-muted-foreground">
            Deterministic extraction from discussion notes and minutes content.
          </p>
        </div>
        {extracted.length > 0 ? (
          <Button size="sm" variant="outline" onClick={() => createAllMutation.mutate()}>
            Create All
          </Button>
        ) : null}
      </div>

      {extracted.length === 0 ? (
        <p className="text-sm text-muted-foreground">No action items were extracted from the minutes.</p>
      ) : (
        extracted.map((action) => {
          const created = existingTitles.has(action.title.trim().toLowerCase());
          return (
            <div key={`${action.title}-${action.assigned_to}`} className="rounded-lg border px-3 py-3">
              <div className="flex items-start justify-between gap-2">
                <div>
                  <p className="font-medium">{action.title}</p>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {action.assigned_to} • {action.due_date ?? 'No due date'}
                  </p>
                </div>
                <Badge variant={created ? 'success' : 'outline'} className="capitalize">
                  {created ? 'Created' : action.priority}
                </Badge>
              </div>
              <p className="mt-2 text-sm text-muted-foreground">{action.description}</p>
              {!created ? (
                <Button className="mt-3" size="sm" variant="outline" onClick={() => setSelected(action)}>
                  Create Action Item
                </Button>
              ) : null}
            </div>
          );
        })
      )}

      <CreateActionItemDialog
        open={Boolean(selected)}
        onOpenChange={(open) => !open && setSelected(null)}
        committees={[committee]}
        meetings={[meeting]}
        preset={
          selected
            ? {
                meeting_id: meeting.id,
                committee_id: committee.id,
                title: selected.title,
                description: selected.description,
                priority: selected.priority,
                due_date: selected.due_date ?? new Date().toISOString().slice(0, 10),
              }
            : undefined
        }
      />
    </div>
  );
}
