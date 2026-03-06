import { Suspense } from 'react';
import type { Metadata } from 'next';
import { ResetPasswordForm } from '@/components/auth/reset-password-form';
import { Spinner } from '@/components/ui/spinner';

export const metadata: Metadata = {
  title: 'Reset Password — Clario 360',
};

export default function ResetPasswordPage() {
  return (
    <Suspense fallback={<div className="flex justify-center py-8"><Spinner /></div>}>
      <ResetPasswordForm />
    </Suspense>
  );
}
