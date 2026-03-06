'use client';

import { useState, useEffect } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { apiGet, apiPut } from '@/lib/api';
import { API_ENDPOINTS } from '@/lib/constants';
import { PageHeader } from '@/components/common/page-header';
import { LoadingSkeleton } from '@/components/common/loading-skeleton';
import { ErrorState } from '@/components/common/error-state';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Switch } from '@/components/ui/switch';
import { Button } from '@/components/ui/button';

interface NotificationPreferences {
  channels: {
    in_app: boolean;
    email: boolean;
    push: boolean;
  };
  quiet_hours: {
    enabled: boolean;
    start: string;
    end: string;
    timezone: string;
  };
  digest: {
    daily: boolean;
    weekly: boolean;
  };
  overrides: Record<string, { email: boolean; push: boolean }>;
}

const NOTIFICATION_TYPES = [
  { key: 'alert.created', label: 'Security Alerts' },
  { key: 'task.assigned', label: 'Task Assignments' },
  { key: 'pipeline.failed', label: 'Pipeline Events' },
  { key: 'data_quality.issue', label: 'Data Quality Issues' },
  { key: 'contract.expiring', label: 'Contract Expirations' },
  { key: 'meeting.reminder', label: 'Meeting Reminders' },
  { key: 'system.maintenance', label: 'System Maintenance' },
];

const DEFAULT_PREFS: NotificationPreferences = {
  channels: { in_app: true, email: true, push: true },
  quiet_hours: { enabled: false, start: '22:00', end: '08:00', timezone: 'UTC' },
  digest: { daily: false, weekly: false },
  overrides: {},
};

export default function NotificationPreferencesPage() {
  const [prefs, setPrefs] = useState<NotificationPreferences>(DEFAULT_PREFS);
  const [isDirty, setIsDirty] = useState(false);

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['notification-preferences'],
    queryFn: () => apiGet<NotificationPreferences>(API_ENDPOINTS.NOTIFICATIONS_PREFERENCES),
  });

  useEffect(() => {
    if (data) {
      setPrefs(data);
      setIsDirty(false);
    }
  }, [data]);

  const saveMutation = useMutation({
    mutationFn: () => apiPut(API_ENDPOINTS.NOTIFICATIONS_PREFERENCES, prefs),
    onSuccess: () => setIsDirty(false),
  });

  const updateChannel = (key: keyof NotificationPreferences['channels'], val: boolean) => {
    if (key === 'in_app') return; // always on
    setPrefs((p) => ({ ...p, channels: { ...p.channels, [key]: val } }));
    setIsDirty(true);
  };

  const updateQuietHours = (key: keyof NotificationPreferences['quiet_hours'], val: boolean | string) => {
    setPrefs((p) => ({ ...p, quiet_hours: { ...p.quiet_hours, [key]: val } }));
    setIsDirty(true);
  };

  const updateDigest = (key: keyof NotificationPreferences['digest'], val: boolean) => {
    setPrefs((p) => ({ ...p, digest: { ...p.digest, [key]: val } }));
    setIsDirty(true);
  };

  if (isLoading) return <LoadingSkeleton variant="card" count={3} />;
  if (isError) return <ErrorState message="Failed to load preferences" onRetry={() => refetch()} />;

  return (
    <div className="space-y-6 max-w-2xl">
      <PageHeader title="Notification Preferences" description="Customize how and when you receive notifications" />

      {/* Channels */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Notification Channels</CardTitle>
          <CardDescription>Choose how you receive notifications</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {[
            { key: 'in_app' as const, label: 'In-app notifications', disabled: true },
            { key: 'email' as const, label: 'Email notifications', disabled: false },
            { key: 'push' as const, label: 'Real-time push', disabled: false },
          ].map((ch) => (
            <div key={ch.key} className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium">{ch.label}</p>
                {ch.disabled && <p className="text-xs text-muted-foreground">Always enabled</p>}
              </div>
              <Switch
                checked={prefs.channels[ch.key]}
                onCheckedChange={(v) => updateChannel(ch.key, v)}
                disabled={ch.disabled}
                aria-label={ch.label}
              />
            </div>
          ))}
        </CardContent>
      </Card>

      {/* Per-type overrides */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Per-Type Overrides</CardTitle>
          <CardDescription>Customize channels for specific notification types</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b">
                  <th className="pb-2 text-left font-medium">Type</th>
                  <th className="pb-2 px-4 text-center font-medium">Email</th>
                  <th className="pb-2 px-4 text-center font-medium">Push</th>
                </tr>
              </thead>
              <tbody>
                {NOTIFICATION_TYPES.map((nt) => {
                  const override = prefs.overrides[nt.key] ?? { email: true, push: true };
                  return (
                    <tr key={nt.key} className="border-b last:border-0">
                      <td className="py-2.5 text-sm">{nt.label}</td>
                      <td className="py-2.5 px-4 text-center">
                        <Switch
                          checked={override.email}
                          onCheckedChange={(v) => {
                            setPrefs((p) => ({
                              ...p,
                              overrides: { ...p.overrides, [nt.key]: { ...override, email: v } },
                            }));
                            setIsDirty(true);
                          }}
                          aria-label={`${nt.label} email`}
                        />
                      </td>
                      <td className="py-2.5 px-4 text-center">
                        <Switch
                          checked={override.push}
                          onCheckedChange={(v) => {
                            setPrefs((p) => ({
                              ...p,
                              overrides: { ...p.overrides, [nt.key]: { ...override, push: v } },
                            }));
                            setIsDirty(true);
                          }}
                          aria-label={`${nt.label} push`}
                        />
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>

      {/* Quiet hours */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Quiet Hours</CardTitle>
          <CardDescription>During quiet hours, only critical notifications are delivered immediately.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <p className="text-sm font-medium">Enable quiet hours</p>
            <Switch
              checked={prefs.quiet_hours.enabled}
              onCheckedChange={(v) => updateQuietHours('enabled', v)}
              aria-label="Enable quiet hours"
            />
          </div>
          {prefs.quiet_hours.enabled && (
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="text-xs text-muted-foreground">Start time</label>
                <input
                  type="time"
                  value={prefs.quiet_hours.start}
                  onChange={(e) => updateQuietHours('start', e.target.value)}
                  className="mt-1 block w-full rounded-md border bg-background px-3 py-1.5 text-sm"
                />
              </div>
              <div>
                <label className="text-xs text-muted-foreground">End time</label>
                <input
                  type="time"
                  value={prefs.quiet_hours.end}
                  onChange={(e) => updateQuietHours('end', e.target.value)}
                  className="mt-1 block w-full rounded-md border bg-background px-3 py-1.5 text-sm"
                />
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Digest */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Digest</CardTitle>
          <CardDescription>Receive a summary of notifications via email at 8:00 AM in your timezone.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <p className="text-sm font-medium">Daily digest</p>
            <Switch checked={prefs.digest.daily} onCheckedChange={(v) => updateDigest('daily', v)} aria-label="Daily digest" />
          </div>
          <div className="flex items-center justify-between">
            <p className="text-sm font-medium">Weekly digest</p>
            <Switch checked={prefs.digest.weekly} onCheckedChange={(v) => updateDigest('weekly', v)} aria-label="Weekly digest" />
          </div>
        </CardContent>
      </Card>

      <div className="flex justify-end">
        <Button
          onClick={() => saveMutation.mutate()}
          disabled={!isDirty || saveMutation.isPending}
        >
          {saveMutation.isPending ? 'Saving...' : 'Save preferences'}
        </Button>
      </div>
    </div>
  );
}
