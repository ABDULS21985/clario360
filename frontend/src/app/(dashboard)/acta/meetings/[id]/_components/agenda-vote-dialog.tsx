'use client';

import { useMemo } from 'react';
import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useForm } from 'react-hook-form';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
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
import { agendaVoteSchema, calculateVoteOutcome, type AgendaVoteFormValues } from '@/lib/enterprise';
import type { ActaAgendaItem } from '@/types/suites';

interface AgendaVoteDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  item: ActaAgendaItem | null;
  presentCount: number;
  onSubmit: (values: AgendaVoteFormValues) => void;
  pending?: boolean;
}

export function AgendaVoteDialog({
  open,
  onOpenChange,
  item,
  presentCount,
  onSubmit,
  pending = false,
}: AgendaVoteDialogProps) {
  const form = useForm<AgendaVoteFormValues>({
    resolver: zodResolver(agendaVoteSchema),
    defaultValues: {
      vote_type: item?.vote_type ?? 'majority',
      votes_for: item?.votes_for ?? 0,
      votes_against: item?.votes_against ?? 0,
      votes_abstained: item?.votes_abstained ?? 0,
      notes: item?.vote_notes ?? '',
    },
  });

  const values = form.watch();
  const totalVotes = values.votes_for + values.votes_against + values.votes_abstained;
  const outcome = useMemo(
    () =>
      calculateVoteOutcome(
        values.vote_type,
        values.votes_for,
        values.votes_against,
        values.votes_abstained,
      ),
    [values.vote_type, values.votes_for, values.votes_against, values.votes_abstained],
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Record Vote</DialogTitle>
          <DialogDescription>
            Capture the voting outcome for {item?.title ?? 'agenda item'}.
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <form
            className="space-y-4"
            onSubmit={form.handleSubmit((vote) => {
              if (totalVotes > presentCount) {
                form.setError('votes_for', {
                  message: `Vote total cannot exceed ${presentCount} present attendees.`,
                });
                return;
              }
              onSubmit(vote);
            })}
          >
            <FormField name="vote_type" label="Vote type" required>
              <Select
                value={values.vote_type}
                onValueChange={(value) =>
                  form.setValue('vote_type', value as AgendaVoteFormValues['vote_type'], {
                    shouldValidate: true,
                  })
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="unanimous">Unanimous</SelectItem>
                  <SelectItem value="majority">Simple Majority</SelectItem>
                  <SelectItem value="two_thirds">Two-thirds Majority</SelectItem>
                  <SelectItem value="roll_call">Roll Call</SelectItem>
                </SelectContent>
              </Select>
            </FormField>

            <div className="grid gap-4 md:grid-cols-3">
              <FormField name="votes_for" label="In favor" required>
                <Input
                  type="number"
                  min={0}
                  value={values.votes_for}
                  onChange={(event) =>
                    form.setValue('votes_for', Number(event.target.value), { shouldValidate: true })
                  }
                />
              </FormField>
              <FormField name="votes_against" label="Against" required>
                <Input
                  type="number"
                  min={0}
                  value={values.votes_against}
                  onChange={(event) =>
                    form.setValue('votes_against', Number(event.target.value), { shouldValidate: true })
                  }
                />
              </FormField>
              <FormField name="votes_abstained" label="Abstained" required>
                <Input
                  type="number"
                  min={0}
                  value={values.votes_abstained}
                  onChange={(event) =>
                    form.setValue('votes_abstained', Number(event.target.value), { shouldValidate: true })
                  }
                />
              </FormField>
            </div>

            <div className="rounded-xl border px-4 py-3 text-sm">
              <p className="font-medium">Result preview</p>
              <p className="mt-1 text-muted-foreground">
                {outcome.label}
                {outcome.result === 'tied' ? ' — Chair may cast deciding vote.' : ''}
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                Total votes: {totalVotes} / {presentCount}
              </p>
            </div>

            <FormField name="notes" label="Notes" required>
              <Textarea {...form.register('notes')} rows={4} />
            </FormField>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={pending}>
                {pending ? 'Saving…' : 'Record vote'}
              </Button>
            </DialogFooter>
          </form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
