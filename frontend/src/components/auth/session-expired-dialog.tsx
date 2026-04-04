'use client';

import { usePathname, useRouter } from 'next/navigation';
import { useAuth } from '@/hooks/use-auth';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Lock } from 'lucide-react';

export function SessionExpiredDialog() {
  const { sessionExpired } = useAuth();
  const pathname = usePathname();
  const router = useRouter();

  const handleSignIn = () => {
    const redirect = encodeURIComponent(pathname ?? '/dashboard');
    router.push(`/login?redirect=${redirect}`);
  };

  return (
    <Dialog
      open={sessionExpired}
      // Non-dismissible: no onOpenChange so Escape/click-outside does nothing
    >
      <DialogContent
        className="sm:max-w-md"
        // Prevent closing by any means
        onPointerDownOutside={(e) => e.preventDefault()}
        onEscapeKeyDown={(e) => e.preventDefault()}
        onInteractOutside={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <div className="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-destructive/10">
            <Lock className="h-6 w-6 text-destructive" />
          </div>
          <DialogTitle className="text-center">Session Expired</DialogTitle>
          <DialogDescription className="text-center">
            Your session has expired due to inactivity. Please sign in again to continue.
          </DialogDescription>
        </DialogHeader>
        <div className="mt-4 flex justify-center">
          <Button onClick={handleSignIn} className="w-full">
            Sign In
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
