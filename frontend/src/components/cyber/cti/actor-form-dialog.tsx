'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { ActorForm } from '@/components/cyber/cti/actor-form';
import { createThreatActor, updateThreatActor } from '@/lib/cti-api';
import type { CreateThreatActorRequest, CTIThreatActor, UpdateThreatActorRequest } from '@/types/cti';

interface ActorFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  actor?: CTIThreatActor | null;
  onSuccess?: (actor: CTIThreatActor) => void;
}

export function ActorFormDialog({
  open,
  onOpenChange,
  actor,
  onSuccess,
}: ActorFormDialogProps) {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: async (values: CreateThreatActorRequest | UpdateThreatActorRequest) => {
      if (actor) {
        return updateThreatActor(actor.id, values as UpdateThreatActorRequest);
      }
      return createThreatActor(values as CreateThreatActorRequest);
    },
    onSuccess: async (savedActor) => {
      await queryClient.invalidateQueries({ queryKey: ['cti-actors'] });
      if (actor) {
        await queryClient.invalidateQueries({ queryKey: ['cti-actor', actor.id] });
      }
      toast.success(actor ? 'Threat actor updated' : 'Threat actor created');
      onOpenChange(false);
      onSuccess?.(savedActor);
    },
    onError: () => {
      toast.error(actor ? 'Failed to update threat actor' : 'Failed to create threat actor');
    },
  });

  const handleSubmit = async (values: CreateThreatActorRequest | UpdateThreatActorRequest) => {
    await mutation.mutateAsync(values);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{actor ? 'Edit Threat Actor' : 'Create Threat Actor'}</DialogTitle>
          <DialogDescription>
            Capture the actor profile, likely motivation, and analyst context used across campaigns.
          </DialogDescription>
        </DialogHeader>
        <ActorForm
          actor={actor}
          onSubmit={handleSubmit}
          onCancel={() => onOpenChange(false)}
          isLoading={mutation.isPending}
        />
      </DialogContent>
    </Dialog>
  );
}
