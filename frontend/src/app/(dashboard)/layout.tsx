'use client';

import { Sidebar } from '@/components/layout/sidebar';
import { Header } from '@/components/layout/header';
import { MobileSidebar } from '@/components/layout/mobile-sidebar';
import { CommandPalette } from '@/components/layout/command-palette';
import { ConnectionBanner } from '@/components/layout/connection-banner';
import { WebSocketProvider } from '@/components/providers/websocket-provider';

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <WebSocketProvider>
      <div className="relative flex h-screen overflow-hidden p-3">
        <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(15,118,110,0.08),transparent_24%),radial-gradient(circle_at_bottom_right,rgba(180,83,9,0.07),transparent_20%)]" />
        <Sidebar />
        <MobileSidebar />
        <div className="relative flex min-w-0 flex-1 flex-col overflow-hidden rounded-[30px] border border-[color:var(--panel-border)] bg-[var(--panel-bg)] shadow-[var(--shell-shadow)] backdrop-blur-xl">
          <ConnectionBanner />
          <Header />
          <main className="flex-1 overflow-auto">
            <div className="w-full px-4 py-6 sm:px-6 lg:px-8 xl:px-10">
              {children}
            </div>
          </main>
        </div>
        <CommandPalette />
      </div>
    </WebSocketProvider>
  );
}
