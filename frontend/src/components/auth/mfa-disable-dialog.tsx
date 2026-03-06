'use client';

import React, { useState } from 'react';
import { ShieldOff, AlertTriangle } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { MFACodeInput } from './mfa-code-input';
import { apiPost } from '@/lib/api';
import { useAuthStore } from '@/stores/auth-store';
import { useToast } from '@/hooks/use-toast';
import { isApiError } from '@/types/api';
import { API_ENDPOINTS } from '@/lib/constants';

interface MFADisableDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function MFADisableDialog({ open, onOpenChange }: MFADisableDialogProps) {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { toast } = useToast();

  const handleDisable = async (code: string) => {
    setIsLoading(true);
    setError(null);
    try {
      await apiPost(API_ENDPOINTS.USERS_ME_MFA_DISABLE, { code });
      useAuthStore.setState((s) => ({
        user: s.user ? { ...s.user, mfa_enabled: false } : s.user,
      }));
      toast({
        title: 'Two-factor authentication disabled',
        description: 'Your account is now less secure. Consider re-enabling MFA.',
      });
      onOpenChange(false);
    } catch (err) {
      setError(
        isApiError(err) ? err.message : 'Invalid code or unable to disable MFA.',
      );
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <div className="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-destructive/10">
            <ShieldOff className="h-6 w-6 text-destructive" />
          </div>
          <DialogTitle className="text-center">Disable Two-Factor Authentication</DialogTitle>
          <DialogDescription className="text-center">
            This will make your account less secure. Are you sure?
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <Alert variant="warning">
            <AlertTriangle className="h-4 w-4" />
            <AlertDescription>
              Disabling MFA removes an important layer of security from your account. Only
              proceed if your authenticator device is lost.
            </AlertDescription>
          </Alert>
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}
          <div className="space-y-2">
            <p className="text-center text-sm text-muted-foreground">
              Enter the 6-digit code from your authenticator app to confirm:
            </p>
            <div className="flex justify-center">
              <MFACodeInput
                onComplete={handleDisable}
                disabled={isLoading}
                error={!!error}
              />
            </div>
          </div>
        </div>
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isLoading}
            className="flex-1"
          >
            Cancel
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
