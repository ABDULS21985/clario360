import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Clario 360 — Sign In',
};

export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="relative flex min-h-screen flex-col items-center justify-center bg-gradient-to-br from-[#0A2E2F] to-[#0D4B4F] px-4">
      {/* Brand mark */}
      <div className="mb-8 text-center">
        <h1 className="text-3xl font-bold tracking-tight text-white">Clario 360</h1>
        <p className="mt-1 text-sm text-[#C6A962]">Enterprise AI Platform</p>
      </div>

      {/* Card */}
      <div className="w-full max-w-md rounded-lg bg-white p-8 shadow-2xl dark:bg-gray-900">
        {children}
      </div>

      {/* Footer */}
      <p className="mt-8 text-xs text-white/50">
        &copy; {new Date().getFullYear()} Clario 360. All rights reserved.
      </p>
    </div>
  );
}
