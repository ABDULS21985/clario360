import type { Metadata } from 'next';
import localFont from 'next/font/local';
import './globals.css';
import { AuthProvider } from '@/components/providers/auth-provider';
import { QueryProvider } from '@/components/providers/query-provider';
import { ToastProvider } from '@/components/providers/toast-provider';
import { SessionExpiredDialog } from '@/components/auth/session-expired-dialog';

// Local font files prevent build failures caused by Google Fonts network timeouts.
// Source files downloaded from Google Fonts (fonts.gstatic.com) and committed to
// the repository so builds are fully offline-capable.
const plusJakartaSans = localFont({
  src: [
    {
      path: '../../public/fonts/plus-jakarta-sans-latin-ext.woff2',
      weight: '200 800',
      style: 'normal',
    },
    {
      path: '../../public/fonts/plus-jakarta-sans-latin.woff2',
      weight: '200 800',
      style: 'normal',
    },
  ],
  variable: '--font-sans',
  display: 'swap',
});

const ibmPlexMono = localFont({
  src: [
    {
      path: '../../public/fonts/ibm-plex-mono-400-latin.woff2',
      weight: '400',
      style: 'normal',
    },
    {
      path: '../../public/fonts/ibm-plex-mono-500-latin.woff2',
      weight: '500',
      style: 'normal',
    },
    {
      path: '../../public/fonts/ibm-plex-mono-600-latin.woff2',
      weight: '600',
      style: 'normal',
    },
  ],
  variable: '--font-mono',
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
