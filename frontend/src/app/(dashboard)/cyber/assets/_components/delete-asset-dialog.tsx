'use client';

import { useState } from 'react';
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
import { useApiMutation } from '@/hooks/use-api-mutation';
import { API_ENDPOINTS } from '@/lib/constants';
import { AlertTriangle } from 'lucide-react';
import type { CyberAsset } from '@/types/cyber';

interface DeleteAssetDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  asset: CyberAsset;
  onSuccess?: () => void;
}

export function DeleteAssetDialog({ open, onOpenChange, asset, onSuccess }: DeleteAssetDialogProps) {
  const [confirmation, setConfirmation] = useState('');

  const { mutate, isPending } = useApiMutation<void, void>(
    'delete',
    `${API_ENDPOINTS.CYBER_ASSETS}/${asset.id}`,
    {
      successMessage: 'Asset deleted',
      invalidateKeys: ['cyber-assets', 'cyber-assets-stats'],
      onSuccess: () => {
        setConfirmation('');
        onOpenChange(false);
        onSuccess?.();
      },
    },
  );

  const handleClose = () => {
    setConfirmation('');
    onOpenChange(false);
  };

  const confirmed = confirmation === 'DELETE';

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-destructive">
            <AlertTriangle className="h-5 w-5" />
            Delete Asset
          </DialogTitle>
          <DialogDescription>
            This action is <strong>irreversible</strong>. All associated vulnerabilities, alerts references, and scan history for <strong>{asset.name}</strong> will be unlinked.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3 rounded-md border border-destructive/30 bg-destructive/5 p-4">
          <div className="text-sm">
            <span className="font-medium">Asset:</span> {asset.name}
          </div>
          <div className="text-sm">
            <span className="font-medium">Type:</span> {asset.type}
          </div>
          <div className="text-sm">
            <span className="font-medium">Criticality:</span> {asset.criticality}
          </div>
          {(asset.vulnerability_count ?? 0) > 0 && (
            <div className="text-sm text-destructive">
              <span className="font-medium">Warning:</span> This asset has {asset.vulnerability_count} open vulnerabilities.
            </div>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="confirm-delete">
            Type <strong>DELETE</strong> to confirm
          </Label>
          <Input
            id="confirm-delete"
            value={confirmation}
            onChange={(e) => setConfirmation(e.target.value)}
            placeholder="DELETE"
            className="font-mono"
          />
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button
            type="button"
            variant="destructive"
            disabled={!confirmed || isPending}
            onClick={() => mutate(undefined as unknown as void)}
          >
            {isPending ? 'Deleting…' : 'Delete Asset'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
