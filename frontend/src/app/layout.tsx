import type { Metadata } from 'next';
import { IBM_Plex_Mono, Plus_Jakarta_Sans } from 'next/font/google';
import './globals.css';
import { AuthProvider } from '@/components/providers/auth-provider';
import { QueryProvider } from '@/components/providers/query-provider';
import { ToastProvider } from '@/components/providers/toast-provider';
import { SessionExpiredDialog } from '@/components/auth/session-expired-dialog';

const plusJakartaSans = Plus_Jakarta_Sans({
  subsets: ['latin'],
  variable: '--font-sans',
  display: 'swap',
});

const ibmPlexMono = IBM_Plex_Mono({
  subsets: ['latin'],
  variable: '--font-mono',
  weight: ['400', '500', '600'],
  display: 'swap',
});

export const metadata: Metadata = {
  title: 'Clario 360 — Enterprise AI Platform',
  description: 'Saudi-owned, Kubernetes-native, multi-suite enterprise AI platform',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className={`${plusJakartaSans.variable} ${ibmPlexMono.variable} antialiased`}>
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
