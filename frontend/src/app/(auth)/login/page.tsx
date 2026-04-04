import { Suspense } from 'react';
import type { Metadata } from 'next';
import { AuthLoadingState } from '@/components/auth/auth-page-primitives';
import { LoginForm } from '@/components/auth/login-form';

export const metadata: Metadata = {
  title: 'Sign In — Clario 360',
};

export default function LoginPage() {
  return (
    <Suspense
      fallback={
        <AuthLoadingState
          label="Loading secure access"
          detail="We are preparing the sign-in experience and syncing the current authentication state."
        />
      }
    >
      <LoginForm />
    </Suspense>
  );
}
