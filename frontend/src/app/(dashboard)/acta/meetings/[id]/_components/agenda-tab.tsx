'use client';

import { useEffect, useState } from 'react';
import { DndContext, PointerSensor, closestCenter, useSensor, useSensors, type DragEndEvent } from '@dnd-kit/core';
import { SortableContext, arrayMove, verticalListSortingStrategy } from '@dnd-kit/sortable';
import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useForm } from 'react-hook-form';
import { Plus } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { FormField } from '@/components/shared/forms/form-field';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { agendaItemSchema, type AgendaItemFormValues } from '@/lib/enterprise';
import type { ActaAgendaItem, ActaMeeting, UserDirectoryEntry } from '@/types/suites';
import { AgendaItemCard } from './agenda-item-card';

interface AgendaTabProps {
  meeting: ActaMeeting;
  items: ActaAgendaItem[];
  presentCount: number;
  users: UserDirectoryEntry[];
  onCreate: (values: AgendaItemFormValues) => void;
  onDelete: (item: ActaAgendaItem) => void;
  onReorder: (itemIds: string[]) => void;
  onRecordVote: (item: ActaAgendaItem) => void;
  onSaveNotes: (itemId: string, notes: string) => void;
}

export function AgendaTab({
  meeting,
  items,
  presentCount,
  users,
  onCreate,
  onDelete,
  onReorder,
  onRecordVote,
  onSaveNotes,
}: AgendaTabProps) {
  const [ordered, setOrdered] = useState(items);
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 4 } }));
  const canReorder = meeting.status === 'draft' || meeting.status === 'scheduled';
  const canAdd = canReorder;
  const form = useForm<AgendaItemFormValues>({
    resolver: zodResolver(agendaItemSchema),
    defaultValues: {
      title: '',
      description: '',
      item_number: '',
      presenter_user_id: null,
      presenter_name: '',
      duration_minutes: 15,
      order_index: null,
      parent_item_id: null,
      requires_vote: false,
      vote_type: null,
      attachment_ids: [],
      category: 'regular',
      confidential: false,
    },
  });

  useEffect(() => {
    setOrdered(items);
  }, [items]);

  const handleDragEnd = (event: DragEndEvent) => {
    if (!canReorder || !event.over || event.active.id === event.over.id) {
      return;
    }
    const oldIndex = ordered.findIndex((item) => item.id === event.active.id);
    const newIndex = ordered.findIndex((item) => item.id === event.over?.id);
    const next = arrayMove(ordered, oldIndex, newIndex);
    setOrdered(next);
    onReorder(next.map((item) => item.id));
  };

  return (
    <div className="space-y-4">
      <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
        <SortableContext items={ordered.map((item) => item.id)} strategy={verticalListSortingStrategy}>
          <div className="space-y-3">
            {ordered.map((item) => (
              <AgendaItemCard
                key={item.id}
                item={item}
                meetingStatus={meeting.status}
                canReorder={canReorder}
                presentCount={presentCount}
                onDelete={onDelete}
                onRecordVote={onRecordVote}
                onSaveNotes={onSaveNotes}
              />
            ))}
          </div>
        </SortableContext>
      </DndContext>

      {canAdd ? (
        <div className="rounded-xl border border-dashed bg-card p-4">
          <p className="mb-4 text-sm font-medium">Add Agenda Item</p>
          <FormProvider {...form}>
            <form
              className="space-y-4"
              onSubmit={form.handleSubmit((values) => onCreate(values))}
            >
              <div className="grid gap-4 md:grid-cols-2">
                <FormField name="title" label="Title" required>
                  <Input {...form.register('title')} />
                </FormField>
                <FormField name="duration_minutes" label="Duration (minutes)" required>
                  <Input
                    type="number"
                    min={5}
                    max={180}
                    value={form.watch('duration_minutes')}
                    onChange={(event) =>
                      form.setValue('duration_minutes', Number(event.target.value), { shouldValidate: true })
                    }
                  />
                </FormField>
              </div>
              <FormField name="description" label="Description" required>
                <Textarea {...form.register('description')} rows={3} />
              </FormField>
              <div className="grid gap-4 md:grid-cols-3">
                <FormField name="presenter_user_id" label="Presenter">
                  <Select
                    value={form.watch('presenter_user_id') ?? undefined}
                    onValueChange={(value) => {
                      const user = users.find((entry) => entry.id === value);
                      form.setValue('presenter_user_id', value, { shouldValidate: true });
                      form.setValue('presenter_name', user ? `${user.first_name} ${user.last_name}`.trim() : '', { shouldValidate: true });
                    }}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Select presenter" />
                    </SelectTrigger>
                    <SelectContent>
                      {users.map((user) => (
                        <SelectItem key={user.id} value={user.id}>
                          {`${user.first_name} ${user.last_name}`.trim()}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </FormField>
                <FormField name="category" label="Category">
                  <Select
                    value={form.watch('category') ?? 'regular'}
                    onValueChange={(value) =>
                      form.setValue('category', value as AgendaItemFormValues['category'], { shouldValidate: true })
                    }
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {['regular', 'special', 'information', 'decision', 'discussion', 'ratification'].map((category) => (
                        <SelectItem key={category} value={category}>
                          {category}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </FormField>
                <FormField name="vote_type" label="Vote type">
                  <Select
                    value={form.watch('vote_type') ?? 'none'}
                    onValueChange={(value) => {
                      if (value === 'none') {
                        form.setValue('requires_vote', false, { shouldValidate: true });
                        form.setValue('vote_type', null, { shouldValidate: true });
                        return;
                      }
                      form.setValue('requires_vote', true, { shouldValidate: true });
                      form.setValue('vote_type', value as AgendaItemFormValues['vote_type'], { shouldValidate: true });
                    }}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="No vote required" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="none">No vote required</SelectItem>
                      <SelectItem value="unanimous">Unanimous</SelectItem>
                      <SelectItem value="majority">Simple majority</SelectItem>
                      <SelectItem value="two_thirds">Two-thirds</SelectItem>
                      <SelectItem value="roll_call">Roll call</SelectItem>
                    </SelectContent>
                  </Select>
                </FormField>
              </div>
              <div className="flex justify-end">
                <Button type="submit">
                  <Plus className="mr-1.5 h-4 w-4" />
                  Add agenda item
                </Button>
              </div>
            </form>
          </FormProvider>
        </div>
      ) : null}
    </div>
  );
}
