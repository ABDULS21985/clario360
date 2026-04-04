'use client';

import { useState } from 'react';
import { toast } from 'sonner';
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
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { apiPut } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';

interface BulkTagDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  assetIds: string[];
  onSuccess?: () => void;
}

export function BulkTagDialog({ open, onOpenChange, assetIds, onSuccess }: BulkTagDialogProps) {
  const [tagInput, setTagInput] = useState('');
  const [tags, setTags] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);

  const handleAddTag = () => {
    const value = tagInput.trim().toLowerCase();
    if (value && !tags.includes(value)) {
      setTags((prev) => [...prev, value]);
    }
    setTagInput('');
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleAddTag();
    }
  };

  const handleRemoveTag = (tag: string) => {
    setTags((prev) => prev.filter((t) => t !== tag));
  };

  const handleSubmit = async () => {
    if (tags.length === 0) {
      toast.error('Add at least one tag');
      return;
    }
    setLoading(true);
    try {
      await apiPut(API_ENDPOINTS.CYBER_ASSETS_BULK_TAGS, {
        asset_ids: assetIds,
        tags,
        action: 'add',
      });
      toast.success(`Tags applied to ${assetIds.length} asset(s)`);
      setTags([]);
      setTagInput('');
      onOpenChange(false);
      onSuccess?.();
    } catch {
      toast.error('Failed to apply tags');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Bulk Tag Management</DialogTitle>
          <DialogDescription>
            Add tags to {assetIds.length} selected asset(s). Tags will be merged with existing tags.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div>
            <Label>Tags to add</Label>
            <div className="flex gap-2 mt-1">
              <Input
                placeholder="Type a tag and press Enter..."
                value={tagInput}
                onChange={(e) => setTagInput(e.target.value)}
                onKeyDown={handleKeyDown}
              />
              <Button type="button" variant="outline" size="sm" onClick={handleAddTag}>
                Add
              </Button>
            </div>
          </div>

          {tags.length > 0 && (
            <div className="flex flex-wrap gap-1">
              {tags.map((tag) => (
                <Badge key={tag} variant="secondary" className="gap-1">
                  {tag}
                  <button
                    type="button"
                    className="ml-1 text-muted-foreground hover:text-foreground"
                    onClick={() => handleRemoveTag(tag)}
                  >
                    &times;
                  </button>
                </Badge>
              ))}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={loading}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={loading || tags.length === 0}>
            {loading ? 'Applying...' : `Apply to ${assetIds.length} Asset(s)`}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
