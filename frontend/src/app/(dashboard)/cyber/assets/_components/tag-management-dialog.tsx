'use client';

import { useState, useEffect } from 'react';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import { X, Plus } from 'lucide-react';
import type { CyberAsset } from '@/types/cyber';

interface TagPatchPayload {
  add?: string[];
  remove?: string[];
}

interface TagManagementDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  asset: CyberAsset;
  onSuccess?: (asset: CyberAsset) => void;
}

export function TagManagementDialog({ open, onOpenChange, asset, onSuccess }: TagManagementDialogProps) {
  const [tags, setTags] = useState<string[]>(asset.tags ?? []);
  const [input, setInput] = useState('');

  // Reset tags when the dialog opens with a (possibly different) asset
  useEffect(() => {
    if (open) {
      setTags(asset.tags ?? []);
    }
  }, [open, asset]);

  const { mutate, isPending } = useApiMutation<CyberAsset, TagPatchPayload>(
    'patch',
    `${API_ENDPOINTS.CYBER_ASSETS}/${asset.id}/tags`,
    {
      successMessage: 'Tags updated',
      invalidateKeys: ['cyber-assets', `cyber-asset-${asset.id}`],
      onSuccess: (updated) => {
        onOpenChange(false);
        onSuccess?.(updated);
      },
    },
  );

  const addTag = () => {
    const trimmed = input.trim().toLowerCase().replace(/\s+/g, '-');
    if (trimmed && !tags.includes(trimmed)) {
      setTags([...tags, trimmed]);
    }
    setInput('');
  };

  const removeTag = (tag: string) => {
    setTags(tags.filter((t) => t !== tag));
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      addTag();
    }
  };

  const handleSave = () => {
    const originalTags = new Set(asset.tags ?? []);
    const currentTags = new Set(tags);
    const add = tags.filter((t) => !originalTags.has(t));
    const remove = (asset.tags ?? []).filter((t) => !currentTags.has(t));

    // Only send non-empty arrays
    const payload: TagPatchPayload = {};
    if (add.length > 0) payload.add = add;
    if (remove.length > 0) payload.remove = remove;

    // Nothing changed — close without a request
    if (!payload.add && !payload.remove) {
      onOpenChange(false);
      return;
    }

    mutate(payload);
  };

  const handleClose = () => {
    setTags(asset.tags ?? []);
    setInput('');
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Manage Tags</DialogTitle>
          <DialogDescription>
            Add or remove tags for <strong>{asset.name}</strong>.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="flex gap-2">
            <Input
              placeholder="Add tag (press Enter)"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              className="flex-1"
            />
            <Button type="button" size="sm" variant="outline" onClick={addTag} disabled={!input.trim()}>
              <Plus className="h-4 w-4" />
            </Button>
          </div>

          <div className="min-h-16 rounded-md border p-3">
            {tags.length === 0 ? (
              <p className="text-xs text-muted-foreground">No tags. Add one above.</p>
            ) : (
              <div className="flex flex-wrap gap-1.5">
                {tags.map((tag) => (
                  <Badge key={tag} variant="secondary" className="gap-1 pr-1">
                    {tag}
                    <button
                      type="button"
                      onClick={() => removeTag(tag)}
                      className="ml-0.5 rounded-sm opacity-70 hover:opacity-100"
                    >
                      <X className="h-3 w-3" />
                    </button>
                  </Badge>
                ))}
              </div>
            )}
          </div>
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button type="button" onClick={handleSave} disabled={isPending}>
            {isPending ? 'Saving…' : 'Save Tags'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
