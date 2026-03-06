'use client';

import { useState } from 'react';
import { Shield, ShieldOff, AlertTriangle } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { MFASetupDialog } from '@/components/auth/mfa-setup-dialog';
import { MFADisableDialog } from '@/components/auth/mfa-disable-dialog';
import { useAuth } from '@/hooks/use-auth';

export function MFASection() {
  const { user, tenant, refreshSession } = useAuth();
  const [setupOpen, setSetupOpen] = useState(false);
  const [disableOpen, setDisableOpen] = useState(false);

  if (!user) return null;

  const mfaRequired = tenant?.settings.mfa_required ?? false;

  const handleMFAChange = async () => {
    await refreshSession();
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Two-Factor Authentication</CardTitle>
        <CardDescription>
          Add an extra layer of security to your account.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {mfaRequired && !user.mfa_enabled && (
          <Alert variant="destructive">
            <AlertTriangle className="h-4 w-4" />
            <AlertDescription>
              Your organization requires two-factor authentication. Please enable it to
              continue using the platform.
            </AlertDescription>
          </Alert>
        )}

        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            {user.mfa_enabled ? (
              <div className="flex h-9 w-9 items-center justify-center rounded-full bg-green-100 dark:bg-green-900/30">
                <Shield className="h-5 w-5 text-green-600 dark:text-green-400" />
              </div>
            ) : (
              <div className="flex h-9 w-9 items-center justify-center rounded-full bg-muted">
                <ShieldOff className="h-5 w-5 text-muted-foreground" />
              </div>
            )}
            <div>
              <p className="text-sm font-medium">
                {user.mfa_enabled ? '2FA is enabled' : '2FA is not enabled'}
              </p>
              <Badge
                variant={user.mfa_enabled ? 'default' : 'outline'}
                className="text-xs mt-0.5"
              >
                {user.mfa_enabled ? 'Enabled' : 'Disabled'}
              </Badge>
            </div>
          </div>

          <div className="flex gap-2">
            {user.mfa_enabled ? (
              <Button
                variant="outline"
                size="sm"
                onClick={() => setDisableOpen(true)}
              >
                Disable 2FA
              </Button>
            ) : (
              <Button size="sm" onClick={() => setSetupOpen(true)}>
                Enable 2FA
              </Button>
            )}
          </div>
        </div>
      </CardContent>

      <MFASetupDialog
        open={setupOpen}
        onOpenChange={(o) => {
          setSetupOpen(o);
          if (!o) handleMFAChange();
        }}
      />

      <MFADisableDialog
        open={disableOpen}
        onOpenChange={(o) => {
          setDisableOpen(o);
          if (!o) handleMFAChange();
        }}
      />
    </Card>
  );
}
