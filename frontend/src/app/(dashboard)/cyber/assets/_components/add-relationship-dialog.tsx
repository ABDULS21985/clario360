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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { apiPost, apiGet } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import type { CyberAsset } from '@/types/cyber';
import type { PaginatedResponse } from '@/types/api';

const RELATIONSHIP_TYPES = [
  { label: 'Hosts', value: 'hosts' },
  { label: 'Runs On', value: 'runs_on' },
  { label: 'Connects To', value: 'connects_to' },
  { label: 'Depends On', value: 'depends_on' },
  { label: 'Managed By', value: 'managed_by' },
  { label: 'Backs Up', value: 'backs_up' },
  { label: 'Load Balances', value: 'load_balances' },
] as const;

interface AddRelationshipDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  asset: CyberAsset;
  onSuccess?: () => void;
}

export function AddRelationshipDialog({ open, onOpenChange, asset, onSuccess }: AddRelationshipDialogProps) {
  const [relationshipType, setRelationshipType] = useState('connects_to');
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<CyberAsset[]>([]);
  const [selectedTarget, setSelectedTarget] = useState<CyberAsset | null>(null);
  const [searching, setSearching] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const handleSearch = async () => {
    if (searchQuery.trim().length < 2) return;
    setSearching(true);
    try {
      const result = await apiGet<PaginatedResponse<CyberAsset>>(API_ENDPOINTS.CYBER_ASSETS, {
        search: searchQuery.trim(),
        per_page: 10,
        page: 1,
      });
      // Exclude the source asset itself
      setSearchResults(result.data.filter((a) => a.id !== asset.id));
    } catch {
      toast.error('Failed to search assets');
    } finally {
      setSearching(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      void handleSearch();
    }
  };

  const handleSubmit = async () => {
    if (!selectedTarget) {
      toast.error('Select a target asset');
      return;
    }
    setSubmitting(true);
    try {
      await apiPost(`${API_ENDPOINTS.CYBER_ASSETS}/${asset.id}/relationships`, {
        target_asset_id: selectedTarget.id,
        relationship_type: relationshipType,
      });
      toast.success(`Relationship created: ${asset.name} → ${selectedTarget.name}`);
      setSearchQuery('');
      setSearchResults([]);
      setSelectedTarget(null);
      onOpenChange(false);
      onSuccess?.();
    } catch {
      toast.error('Failed to create relationship');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Add Relationship</DialogTitle>
          <DialogDescription>
            Create a dependency or connection from <strong>{asset.name}</strong> to another asset.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div>
            <Label>Relationship Type</Label>
            <Select value={relationshipType} onValueChange={setRelationshipType}>
              <SelectTrigger className="mt-1">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {RELATIONSHIP_TYPES.map((rt) => (
                  <SelectItem key={rt.value} value={rt.value}>
                    {rt.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div>
            <Label>Target Asset</Label>
            <div className="flex gap-2 mt-1">
              <Input
                placeholder="Search by name, hostname, or IP..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onKeyDown={handleKeyDown}
              />
              <Button type="button" variant="outline" size="sm" onClick={handleSearch} disabled={searching}>
                {searching ? 'Searching...' : 'Search'}
              </Button>
            </div>
          </div>

          {/* Search results */}
          {searchResults.length > 0 && (
            <div className="max-h-40 overflow-y-auto rounded-md border">
              {searchResults.map((result) => (
                <button
                  key={result.id}
                  type="button"
                  className={`flex w-full items-center gap-3 px-3 py-2 text-left text-sm hover:bg-muted ${
                    selectedTarget?.id === result.id ? 'bg-muted ring-1 ring-primary' : ''
                  }`}
                  onClick={() => setSelectedTarget(result)}
                >
                  <div className="min-w-0 flex-1">
                    <p className="truncate font-medium">{result.name}</p>
                    <p className="text-xs text-muted-foreground">
                      {result.type} · {result.ip_address ?? result.hostname ?? 'no address'}
                    </p>
                  </div>
                </button>
              ))}
            </div>
          )}

          {selectedTarget && (
            <div className="rounded-md border bg-muted/50 p-3 text-sm">
              <p className="font-medium">{asset.name}</p>
              <p className="text-xs text-muted-foreground">
                — {RELATIONSHIP_TYPES.find((r) => r.value === relationshipType)?.label ?? relationshipType} →
              </p>
              <p className="font-medium">{selectedTarget.name}</p>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={submitting}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={submitting || !selectedTarget}>
            {submitting ? 'Creating...' : 'Create Relationship'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
