'use client';

import { useEffect, useRef, useState } from 'react';
import { useSortable } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { ChevronDown, GripVertical, Trash2, Vote } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { agendaVoteSummary } from '@/lib/enterprise';
import { useDebounce } from '@/hooks/use-debounce';
import type { ActaAgendaItem, ActaMeetingStatus } from '@/types/suites';

interface AgendaItemCardProps {
  item: ActaAgendaItem;
  meetingStatus: ActaMeetingStatus;
  canReorder: boolean;
  presentCount: number;
  onDelete: (item: ActaAgendaItem) => void;
  onRecordVote: (item: ActaAgendaItem) => void;
  onSaveNotes: (itemId: string, notes: string) => void;
}

export function AgendaItemCard({
  item,
  meetingStatus,
  canReorder,
  onDelete,
  onRecordVote,
  onSaveNotes,
}: AgendaItemCardProps) {
  const [expanded, setExpanded] = useState(false);
  const [notes, setNotes] = useState(item.notes ?? '');
  const initializedRef = useRef(false);
  const debouncedNotes = useDebounce(notes, 500);
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: item.id, disabled: !canReorder });

  useEffect(() => {
    setNotes(item.notes ?? '');
  }, [item.notes]);

  useEffect(() => {
    if (meetingStatus !== 'in_progress') {
      return;
    }
    if (!initializedRef.current) {
      initializedRef.current = true;
      return;
    }
    if ((item.notes ?? '') !== debouncedNotes) {
      onSaveNotes(item.id, debouncedNotes);
    }
  }, [debouncedNotes, item.id, item.notes, meetingStatus, onSaveNotes]);

  return (
    <div
      ref={setNodeRef}
      style={{ transform: CSS.Transform.toString(transform), transition }}
      className={`rounded-xl border bg-card ${isDragging ? 'opacity-60' : ''}`}
    >
      <div className="flex items-start gap-3 px-4 py-4">
        <button
          type="button"
          className={`mt-1 text-muted-foreground ${canReorder ? 'cursor-grab' : 'cursor-default opacity-40'}`}
          {...attributes}
          {...listeners}
          disabled={!canReorder}
          aria-label="Reorder agenda item"
        >
          <GripVertical className="h-4 w-4" />
        </button>
        <div className="min-w-0 flex-1">
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <p className="truncate font-medium">
                  {item.item_number ? `${item.item_number} ` : ''}
                  {item.title}
                </p>
                {item.category ? (
                  <Badge variant="outline" className="capitalize">
                    {item.category}
                  </Badge>
                ) : null}
                <Badge variant="outline">{item.duration_minutes} min</Badge>
              </div>
              <p className="mt-1 text-xs text-muted-foreground">
                Presenter: {item.presenter_name ?? 'Unassigned'}
              </p>
            </div>
            <div className="flex items-center gap-2">
              <Badge variant="outline" className="capitalize">
                {item.status.replace(/_/g, ' ')}
              </Badge>
              <Button variant="ghost" size="icon" onClick={() => setExpanded((value) => !value)}>
                <ChevronDown className={`h-4 w-4 transition ${expanded ? 'rotate-180' : ''}`} />
              </Button>
            </div>
          </div>
          {expanded ? (
            <div className="mt-4 space-y-4 border-t pt-4">
              <div>
                <p className="text-sm font-medium">Description</p>
                <p className="mt-1 whitespace-pre-wrap text-sm text-muted-foreground">{item.description}</p>
              </div>
              <div>
                <div className="mb-2 flex items-center justify-between">
                  <p className="text-sm font-medium">Discussion Notes</p>
                  <span className="text-xs text-muted-foreground">
                    {meetingStatus === 'in_progress' ? 'Autosaves after 500ms' : 'Read only'}
                  </span>
                </div>
                <Textarea
                  value={notes}
                  onChange={(event) => setNotes(event.target.value)}
                  rows={6}
                  disabled={meetingStatus !== 'in_progress'}
                  placeholder="Capture discussion notes during the meeting."
                />
              </div>
              <div className="flex flex-wrap items-center gap-2">
                {item.requires_vote ? (
                  <Button variant="outline" size="sm" onClick={() => onRecordVote(item)}>
                    <Vote className="mr-1.5 h-4 w-4" />
                    Record Vote
                  </Button>
                ) : null}
                {meetingStatus === 'draft' || meetingStatus === 'scheduled' ? (
                  <Button variant="ghost" size="sm" className="text-rose-600" onClick={() => onDelete(item)}>
                    <Trash2 className="mr-1.5 h-4 w-4" />
                    Remove
                  </Button>
                ) : null}
              </div>
              {agendaVoteSummary(item) ? (
                <div className="rounded-xl border bg-muted/20 px-4 py-3 text-sm">
                  <p className="font-medium">Voting</p>
                  <p className="mt-1 text-muted-foreground">{agendaVoteSummary(item)}</p>
                </div>
              ) : null}
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
}
