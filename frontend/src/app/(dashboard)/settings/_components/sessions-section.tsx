'use client';

import { useState } from 'react';
import { Monitor, Smartphone, Tablet } from 'lucide-react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { RelativeTime } from '@/components/shared/relative-time';
import { apiGet, apiDelete } from '@/lib/api';
import { isApiError } from '@/types/api';

interface Session {
  id: string;
  user_agent: string;
  ip_address: string;
  last_active_at: string;
  created_at: string;
  is_current: boolean;
}

function parseDevice(userAgent: string): { icon: typeof Monitor; label: string } {
  const ua = userAgent.toLowerCase();
  if (ua.includes('mobile') || ua.includes('android') || ua.includes('iphone')) {
    return { icon: Smartphone, label: 'Mobile' };
  }
  if (ua.includes('tablet') || ua.includes('ipad')) {
    return { icon: Tablet, label: 'Tablet' };
  }
  return { icon: Monitor, label: 'Desktop' };
}

function parseBrowser(userAgent: string): string {
  if (userAgent.includes('Firefox')) return 'Firefox';
  if (userAgent.includes('Edg')) return 'Edge';
  if (userAgent.includes('Chrome')) return 'Chrome';
  if (userAgent.includes('Safari')) return 'Safari';
  return 'Browser';
}

export function SessionsSection() {
  const queryClient = useQueryClient();
  const [revokeAllOpen, setRevokeAllOpen] = useState(false);

  const { data: sessions, isLoading } = useQuery<Session[]>({
    queryKey: ['my-sessions'],
    queryFn: () => apiGet<Session[]>('/api/v1/users/me/sessions'),
  });

  const refetch = () => queryClient.invalidateQueries({ queryKey: ['my-sessions'] });

  const handleRevoke = async (sessionId: string) => {
    try {
      await apiDelete(`/api/v1/users/me/sessions/${sessionId}`);
      toast.success('Session revoked.');
      refetch();
    } catch (err) {
      const msg = isApiError(err) ? err.message : 'Failed to revoke session.';
      toast.error(msg);
    }
  };

  const handleRevokeAll = async () => {
    await apiDelete('/api/v1/users/me/sessions?exclude_current=true');
    toast.success('All other sessions have been revoked.');
    refetch();
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Active Sessions</CardTitle>
        <CardDescription>
          Manage your active login sessions across all devices.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        {isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-14 w-full" />
            ))}
          </div>
        ) : !sessions || sessions.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-4">
            Session management is being set up for your organization.
          </p>
        ) : (
          <>
            <div className="space-y-2">
              {sessions.map((session) => {
                const device = parseDevice(session.user_agent);
                const DeviceIcon = device.icon;
                const browser = parseBrowser(session.user_agent);
                return (
                  <div
                    key={session.id}
                    className="flex items-center gap-3 rounded-md border px-3 py-2.5"
                  >
                    <DeviceIcon className="h-5 w-5 text-muted-foreground shrink-0" />
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <p className="text-sm font-medium">{browser} / {device.label}</p>
                        {session.is_current && (
                          <Badge variant="outline" className="text-xs">Current</Badge>
                        )}
                      </div>
                      <p className="text-xs text-muted-foreground">
                        {session.ip_address} ·{' '}
                        <RelativeTime date={session.last_active_at} />
                      </p>
                    </div>
                    {!session.is_current && (
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-xs h-7"
                        onClick={() => handleRevoke(session.id)}
                      >
                        Revoke
                      </Button>
                    )}
                  </div>
                );
              })}
            </div>

            {sessions.filter((s) => !s.is_current).length > 0 && (
              <div className="flex justify-end">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setRevokeAllOpen(true)}
                >
                  Revoke All Other Sessions
                </Button>
              </div>
            )}
          </>
        )}
      </CardContent>

      <ConfirmDialog
        open={revokeAllOpen}
        onOpenChange={setRevokeAllOpen}
        title="Revoke All Other Sessions"
        description="This will log you out of all other devices. Your current session will remain active."
        confirmLabel="Revoke All"
        variant="destructive"
        onConfirm={handleRevokeAll}
      />
    </Card>
  );
}
