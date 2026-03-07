'use client';

import { useState } from 'react';
import ReactMarkdown from 'react-markdown';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { MinutesEditor } from './minutes-editor';
import { AiActionsSidebar } from './ai-actions-sidebar';
import type { ActaActionItem, ActaCommittee, ActaMeeting, ActaMeetingMinutes } from '@/types/suites';

interface MinutesTabProps {
  meeting: ActaMeeting;
  committee: ActaCommittee;
  minutes: ActaMeetingMinutes | null;
  versions: ActaMeetingMinutes[];
  actionItems: ActaActionItem[];
  canApprove: boolean;
  pending?: boolean;
  onGenerate: () => void;
  onSave: (content: string) => void;
  onSubmitReview: () => void;
  onApprove: () => void;
  onPublish: () => void;
}

const WORKFLOW: Array<ActaMeetingMinutes['status']> = ['draft', 'review', 'approved', 'published'];

export function MinutesTab({
  meeting,
  committee,
  minutes,
  versions,
  actionItems,
  canApprove,
  pending = false,
  onGenerate,
  onSave,
  onSubmitReview,
  onApprove,
  onPublish,
}: MinutesTabProps) {
  const [editing, setEditing] = useState(false);

  if (!minutes) {
    return (
      <div className="rounded-xl border bg-card p-8 text-center">
        <p className="text-lg font-semibold">No minutes yet</p>
        <p className="mt-2 text-sm text-muted-foreground">
          Generate deterministic minutes from attendance, agenda notes, votes, and action items.
        </p>
        <Button className="mt-4" onClick={onGenerate} disabled={pending}>
          {pending ? 'Generating minutes…' : 'Generate AI Minutes'}
        </Button>
      </div>
    );
  }

  return (
    <div className="grid gap-4 xl:grid-cols-[1.5fr_0.8fr]">
      <div className="space-y-4">
        <div className="rounded-xl border bg-card p-4">
          <div className="flex flex-wrap items-center gap-2">
            {WORKFLOW.map((step) => (
              <Badge
                key={step}
                variant={minutes.status === step ? 'default' : 'outline'}
                className="capitalize"
              >
                {step.replace(/_/g, ' ')}
              </Badge>
            ))}
            <Badge variant="outline">v{minutes.version}</Badge>
          </div>
          <div className="mt-4 flex flex-wrap gap-2">
            {(minutes.status === 'draft' || minutes.status === 'revision_requested') && !editing ? (
              <Button variant="outline" onClick={() => setEditing(true)}>
                Edit
              </Button>
            ) : null}
            {minutes.status === 'draft' ? (
              <Button onClick={onSubmitReview} disabled={pending}>
                Submit for Review
              </Button>
            ) : null}
            {minutes.status === 'review' ? (
              <Button onClick={onApprove} disabled={!canApprove || pending}>
                Approve
              </Button>
            ) : null}
            {minutes.status === 'approved' ? (
              <Button onClick={onPublish} disabled={pending}>
                Publish
              </Button>
            ) : null}
            {!canApprove && minutes.status === 'review' ? (
              <p className="text-xs text-muted-foreground">Only the committee chair can approve.</p>
            ) : null}
          </div>
        </div>

        {editing ? (
          <MinutesEditor
            initialValue={minutes.content}
            onSave={(content) => {
              onSave(content);
              setEditing(false);
            }}
            onCancel={() => setEditing(false)}
            pending={pending}
          />
        ) : (
          <div className="rounded-xl border bg-card p-4">
            {minutes.ai_summary ? (
              <div className="mb-4 rounded-lg border bg-muted/20 px-4 py-3 text-sm text-muted-foreground">
                {minutes.ai_summary}
              </div>
            ) : null}
            <article className="prose prose-sm max-w-none dark:prose-invert">
              <ReactMarkdown>{minutes.content}</ReactMarkdown>
            </article>
          </div>
        )}

        <div className="rounded-xl border bg-card p-4">
          <p className="font-medium">Version history</p>
          <div className="mt-3 flex flex-wrap gap-2">
            {versions.map((version) => (
              <Badge key={version.id} variant="outline">
                v{version.version}
              </Badge>
            ))}
          </div>
        </div>
      </div>

      <AiActionsSidebar
        meeting={meeting}
        committee={committee}
        extracted={minutes.ai_action_items}
        existingItems={actionItems}
      />
    </div>
  );
}
