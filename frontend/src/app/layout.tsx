import type { Metadata } from 'next';
import './globals.css';
import { AuthProvider } from '@/components/providers/auth-provider';
import { QueryProvider } from '@/components/providers/query-provider';
import { ToastProvider } from '@/components/providers/toast-provider';
import { SessionExpiredDialog } from '@/components/auth/session-expired-dialog';

export const metadata: Metadata = {
  title: 'Clario 360 — Enterprise AI Platform',
  description: 'Saudi-owned, Kubernetes-native, multi-suite enterprise AI platform',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body>
        <QueryProvider>
          <AuthProvider>
            {children}
            <SessionExpiredDialog />
            <ToastProvider />
          </AuthProvider>
        </QueryProvider>
      </body>
    </html>
  );
}
