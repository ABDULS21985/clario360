import { Suspense } from 'react';
import type { Metadata } from 'next';
import { AuthLoadingState } from '@/components/auth/auth-page-primitives';
import { ResetPasswordForm } from '@/components/auth/reset-password-form';

export const metadata: Metadata = {
  title: 'Reset Password — Clario 360',
};

export default function ResetPasswordPage() {
  return (
    <Suspense
      fallback={
        <AuthLoadingState
          label="Loading password reset"
          detail="We are validating the reset flow and preparing the credential update experience."
        />
      }
    >
      <ResetPasswordForm />
    </Suspense>
  );
}
