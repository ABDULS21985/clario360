'use client';

import React, { useState, useEffect, useCallback } from 'react';
import QRCode from 'qrcode';
import { Copy, Download, AlertTriangle } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Label } from '@/components/ui/label';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Spinner } from '@/components/ui/spinner';
import { MFACodeInput } from './mfa-code-input';
import { apiPost } from '@/lib/api';
import { useAuthStore } from '@/stores/auth-store';
import { useToast } from '@/hooks/use-toast';
import { copyToClipboard, downloadTextFile } from '@/lib/utils';
import { API_ENDPOINTS } from '@/lib/constants';
import type { EnableMFAResponse } from '@/types/auth';
import type { ApiError } from '@/types/api';
import { isApiError } from '@/types/api';

type Step = 'qr' | 'verify' | 'recovery';

interface MFASetupDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function MFASetupDialog({ open, onOpenChange }: MFASetupDialogProps) {
  const [step, setStep] = useState<Step>('qr');
  const [mfaData, setMfaData] = useState<EnableMFAResponse | null>(null);
  const [qrDataUrl, setQrDataUrl] = useState<string>('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [manualKeyOpen, setManualKeyOpen] = useState(false);
  const [savedConfirmed, setSavedConfirmed] = useState(false);
  const { toast } = useToast();

  const setUser = useAuthStore((s) => s.user);

  const fetchMFASetup = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const data = await apiPost<EnableMFAResponse>(API_ENDPOINTS.USERS_ME_MFA_ENABLE);
      setMfaData(data);
      const url = await QRCode.toDataURL(data.totp_uri, {
        width: 200,
        margin: 2,
        type: 'image/png',
      });
      setQrDataUrl(url);
    } catch (err) {
      setError(isApiError(err) ? err.message : 'Failed to initialize MFA setup');
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    if (open && step === 'qr') {
      void fetchMFASetup();
    }
  }, [open, step, fetchMFASetup]);

  const handleVerify = async (code: string) => {
    setIsLoading(true);
    setError(null);
    try {
      await apiPost(API_ENDPOINTS.USERS_ME_MFA_VERIFY_SETUP, { code });
      setStep('recovery');
    } catch (err) {
      setError(isApiError(err) ? err.message : 'Invalid code. Please try again.');
    } finally {
      setIsLoading(false);
    }
  };

  const handleCopyManualKey = async () => {
    if (!mfaData) return;
    const success = await copyToClipboard(mfaData.manual_key);
    if (success) toast({ title: 'Copied!', variant: 'success' });
  };

  const handleCopyAllCodes = async () => {
    if (!mfaData) return;
    const text = mfaData.recovery_codes.join('\n');
    const success = await copyToClipboard(text);
    if (success) toast({ title: 'Recovery codes copied!', variant: 'success' });
  };

  const handleDownloadCodes = () => {
    if (!mfaData) return;
    const content = [
      'Clario 360 Recovery Codes',
      '========================',
      '',
      'Keep these codes in a safe place. Each code can only be used once.',
      '',
      ...mfaData.recovery_codes,
    ].join('\n');
    downloadTextFile(content, 'clario360-recovery-codes.txt');
  };

  const handleDone = () => {
    // Update user MFA status in store
    useAuthStore.setState((s) => ({
      user: s.user ? { ...s.user, mfa_enabled: true } : s.user,
    }));
    onOpenChange(false);
    // Reset state for next time
    setTimeout(() => {
      setStep('qr');
      setMfaData(null);
      setQrDataUrl('');
      setSavedConfirmed(false);
    }, 300);
  };

  const handleClose = (open: boolean) => {
    if (!open && step !== 'recovery') {
      onOpenChange(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-md">
        {step === 'qr' && (
          <>
            <DialogHeader>
              <DialogTitle>Set up two-factor authentication</DialogTitle>
              <DialogDescription>
                Scan this QR code with your authenticator app (Google Authenticator, Authy, etc.)
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4">
              {isLoading && (
                <div className="flex justify-center py-8">
                  <Spinner size="lg" />
                </div>
              )}
              {error && (
                <Alert variant="destructive">
                  <AlertDescription>{error}</AlertDescription>
                </Alert>
              )}
              {qrDataUrl && (
                <div className="flex flex-col items-center gap-4">
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img
                    src={qrDataUrl}
                    alt="MFA QR code — scan with your authenticator app"
                    className="rounded-lg border p-2"
                    width={200}
                    height={200}
                  />
                  <button
                    type="button"
                    className="text-sm text-muted-foreground underline-offset-4 hover:underline"
                    onClick={() => setManualKeyOpen((prev) => !prev)}
                    aria-expanded={manualKeyOpen}
                  >
                    Can&apos;t scan the code?
                  </button>
                  {manualKeyOpen && mfaData && (
                    <div className="w-full rounded-md bg-muted p-3">
                      <p className="mb-1 text-xs text-muted-foreground">
                        Enter this key manually:
                      </p>
                      <div className="flex items-center gap-2">
                        <code className="flex-1 break-all font-mono text-sm">
                          {mfaData.manual_key}
                        </code>
                        <button
                          type="button"
                          onClick={handleCopyManualKey}
                          aria-label="Copy manual key"
                          className="shrink-0 rounded p-1 hover:bg-muted-foreground/20"
                        >
                          <Copy className="h-4 w-4" />
                        </button>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>
            <DialogFooter>
              <Button
                onClick={() => setStep('verify')}
                disabled={!qrDataUrl || isLoading}
                className="w-full"
              >
                Next
              </Button>
            </DialogFooter>
          </>
        )}

        {step === 'verify' && (
          <>
            <DialogHeader>
              <DialogTitle>Verify your authenticator</DialogTitle>
              <DialogDescription>
                Enter the 6-digit code from your authenticator app to confirm setup.
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4">
              {error && (
                <Alert variant="destructive">
                  <AlertDescription>{error}</AlertDescription>
                </Alert>
              )}
              <div className="flex justify-center">
                <MFACodeInput
                  onComplete={handleVerify}
                  disabled={isLoading}
                  error={!!error}
                />
              </div>
              {isLoading && (
                <div className="flex justify-center">
                  <Spinner />
                </div>
              )}
            </div>
            <DialogFooter className="flex-row gap-2">
              <Button
                variant="outline"
                onClick={() => { setStep('qr'); setError(null); }}
                className="flex-1"
              >
                Back
              </Button>
            </DialogFooter>
          </>
        )}

        {step === 'recovery' && mfaData && (
          <>
            <DialogHeader>
              <DialogTitle>Save your recovery codes</DialogTitle>
            </DialogHeader>
            <div className="space-y-4">
              <Alert variant="warning">
                <AlertTriangle className="h-4 w-4" />
                <AlertDescription>
                  These codes can be used to access your account if you lose your authenticator
                  device. <strong>Each code can only be used once.</strong> Store them securely —
                  they will not be shown again.
                </AlertDescription>
              </Alert>

              {/* Recovery codes grid */}
              <div className="grid grid-cols-2 gap-2 rounded-md bg-muted p-4">
                {mfaData.recovery_codes.map((code) => (
                  <code key={code} className="text-center font-mono text-sm">
                    {code}
                  </code>
                ))}
              </div>

              {/* Action buttons */}
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleDownloadCodes}
                  className="flex-1"
                >
                  <Download className="mr-2 h-4 w-4" />
                  Download
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleCopyAllCodes}
                  className="flex-1"
                >
                  <Copy className="mr-2 h-4 w-4" />
                  Copy all
                </Button>
              </div>

              {/* Confirmation checkbox */}
              <div className="flex items-start gap-3">
                <Checkbox
                  id="saved-confirmed"
                  checked={savedConfirmed}
                  onCheckedChange={(checked) => setSavedConfirmed(!!checked)}
                />
                <Label htmlFor="saved-confirmed" className="cursor-pointer text-sm leading-relaxed">
                  I have saved my recovery codes in a secure location
                </Label>
              </div>
            </div>
            <DialogFooter>
              <Button onClick={handleDone} disabled={!savedConfirmed} className="w-full">
                Done
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
