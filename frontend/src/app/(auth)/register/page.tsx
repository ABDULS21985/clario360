import { Suspense } from 'react';
import type { Metadata } from 'next';
import { RegisterForm } from '@/components/auth/register-form';
import { Spinner } from '@/components/ui/spinner';

export const metadata: Metadata = {
  title: 'Create Account — Clario 360',
};

export default function RegisterPage() {
  return (
    <Suspense fallback={<div className="flex justify-center py-8"><Spinner /></div>}>
      <RegisterForm />
    </Suspense>
  );
}
