'use client';

import { useState } from 'react';
import ReactMarkdown from 'react-markdown';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Textarea } from '@/components/ui/textarea';
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
  onCreate: (content: string) => void;
  onSave: (content: string) => void;
  onSubmitReview: () => void;
  onRequestRevision: (notes: string) => void;
  onApprove: () => void;
  onPublish: () => void;
}

const WORKFLOW: Array<ActaMeetingMinutes['status']> = ['draft', 'review', 'revision_requested', 'approved', 'published'];

export function MinutesTab({
  meeting,
  committee,
  minutes,
  versions,
  actionItems,
  canApprove,
  pending = false,
  onGenerate,
  onCreate,
  onSave,
  onSubmitReview,
  onRequestRevision,
  onApprove,
  onPublish,
}: MinutesTabProps) {
  const [editing, setEditing] = useState(false);
  const [creatingManual, setCreatingManual] = useState(false);
  const [revisionNotes, setRevisionNotes] = useState('');
  const [showRevisionForm, setShowRevisionForm] = useState(false);

  if (!minutes) {
    if (creatingManual) {
      return (
        <div className="space-y-4">
          <MinutesEditor
            initialValue=""
            onSave={(content) => {
              onCreate(content);
              setCreatingManual(false);
            }}
            onCancel={() => setCreatingManual(false)}
            pending={pending}
          />
        </div>
      );
    }
    return (
      <div className="rounded-xl border bg-card p-8 text-center">
        <p className="text-lg font-semibold">No minutes yet</p>
        <p className="mt-2 text-sm text-muted-foreground">
          Generate deterministic minutes from attendance, agenda notes, votes, and action items, or write them manually.
        </p>
        <div className="mt-4 flex justify-center gap-2">
          <Button onClick={onGenerate} disabled={pending}>
            {pending ? 'Generating minutes…' : 'Generate AI Minutes'}
          </Button>
          <Button variant="outline" onClick={() => setCreatingManual(true)} disabled={pending}>
            Write Manually
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 gap-4 xl:grid-cols-[1.5fr_0.8fr]">
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
            {(minutes.status === 'draft' || minutes.status === 'revision_requested') ? (
              <Button onClick={onSubmitReview} disabled={pending}>
                Submit for Review
              </Button>
            ) : null}
            {minutes.status === 'review' && canApprove ? (
              <>
                <Button onClick={onApprove} disabled={pending}>
                  Approve
                </Button>
                <Button
                  variant="outline"
                  onClick={() => setShowRevisionForm(!showRevisionForm)}
                  disabled={pending}
                >
                  Request Revision
                </Button>
              </>
            ) : null}
            {minutes.status === 'approved' ? (
              <Button onClick={onPublish} disabled={pending}>
                Publish
              </Button>
            ) : null}
            {!canApprove && minutes.status === 'review' ? (
              <p className="text-xs text-muted-foreground">Only the committee chair can approve or request revisions.</p>
            ) : null}
          </div>
          {showRevisionForm && minutes.status === 'review' ? (
            <div className="mt-4 space-y-2 rounded-lg border p-3">
              <Textarea
                rows={3}
                placeholder="Describe what needs to be revised…"
                value={revisionNotes}
                onChange={(event) => setRevisionNotes(event.target.value)}
              />
              <div className="flex gap-2">
                <Button
                  size="sm"
                  disabled={pending || revisionNotes.trim().length < 5}
                  onClick={() => {
                    onRequestRevision(revisionNotes.trim());
                    setRevisionNotes('');
                    setShowRevisionForm(false);
                  }}
                >
                  Send Revision Request
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => {
                    setShowRevisionForm(false);
                    setRevisionNotes('');
                  }}
                >
                  Cancel
                </Button>
              </div>
            </div>
          ) : null}
          {minutes.status === 'revision_requested' && minutes.review_notes ? (
            <div className="mt-4 rounded-lg border border-orange-200 bg-orange-50 px-4 py-3 text-sm dark:border-orange-900 dark:bg-orange-950">
              <p className="font-medium text-orange-800 dark:text-orange-200">Revision requested</p>
              <p className="mt-1 text-orange-700 dark:text-orange-300">{minutes.review_notes}</p>
            </div>
          ) : null}
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
