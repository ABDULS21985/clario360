'use client';

import { useEffect, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { QuorumIndicator } from './quorum-indicator';
import type { ActaAttendee, ActaMeeting } from '@/types/suites';

interface AttendanceTabProps {
  meeting: ActaMeeting;
  attendance: ActaAttendee[];
  currentUserId?: string | null;
  onSaveAttendance: (values: {
    user_id: string;
    status: 'present' | 'absent' | 'proxy' | 'excused';
    proxy_user_name?: string | null;
    proxy_authorized_by?: string | null;
  }) => void;
  onBulkAbsent: (attendance: Array<{
    user_id: string;
    status: 'present' | 'absent' | 'proxy' | 'excused';
    proxy_user_name?: string | null;
    proxy_authorized_by?: string | null;
  }>) => void;
}

export function AttendanceTab({
  meeting,
  attendance,
  currentUserId,
  onSaveAttendance,
  onBulkAbsent,
}: AttendanceTabProps) {
  return (
    <div className="space-y-4">
      <QuorumIndicator attendance={attendance} quorumRequired={meeting.quorum_required} />
      <div className="flex justify-end">
        {(() => {
          const remaining = attendance.filter(
            (entry) => !['present', 'proxy', 'excused', 'absent'].includes(entry.status),
          );
          return (
            <Button
              variant="outline"
              disabled={remaining.length === 0}
              onClick={() =>
                onBulkAbsent(remaining.map((entry) => ({ user_id: entry.user_id, status: 'absent' as const })))
              }
            >
              Mark All Remaining as Absent
              {remaining.length > 0 ? ` (${remaining.length})` : ''}
            </Button>
          );
        })()}
      </div>
      <div className="space-y-3">
        {attendance.map((entry) => (
          <AttendanceRow
            key={entry.id}
            attendee={entry}
            currentUserId={currentUserId}
            onSave={onSaveAttendance}
          />
        ))}
      </div>
    </div>
  );
}

function AttendanceRow({
  attendee,
  currentUserId,
  onSave,
}: {
  attendee: ActaAttendee;
  currentUserId?: string | null;
  onSave: (values: {
    user_id: string;
    status: 'present' | 'absent' | 'proxy' | 'excused';
    proxy_user_name?: string | null;
    proxy_authorized_by?: string | null;
  }) => void;
}) {
  const [status, setStatus] = useState<'present' | 'absent' | 'proxy' | 'excused'>(
    normalizeStatus(attendee.status),
  );
  const [proxyName, setProxyName] = useState(attendee.proxy_user_name ?? '');

  useEffect(() => {
    setStatus(normalizeStatus(attendee.status));
    setProxyName(attendee.proxy_user_name ?? '');
  }, [attendee.proxy_user_name, attendee.status]);

  return (
    <div className="grid grid-cols-1 gap-3 rounded-xl border bg-card px-4 py-3 md:grid-cols-[1.1fr_0.9fr_0.9fr_auto] md:items-center">
      <div>
        <p className="font-medium">{attendee.user_name}</p>
        <p className="text-xs text-muted-foreground">
          {attendee.user_email} • {attendee.member_role.replace(/_/g, ' ')}
        </p>
      </div>
      <Select value={status} onValueChange={(value) => setStatus(value as typeof status)}>
        <SelectTrigger>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="present">Present</SelectItem>
          <SelectItem value="absent">Absent</SelectItem>
          <SelectItem value="proxy">Proxy</SelectItem>
          <SelectItem value="excused">Excused</SelectItem>
        </SelectContent>
      </Select>
      {status === 'proxy' ? (
        <Input value={proxyName} onChange={(event) => setProxyName(event.target.value)} placeholder="Proxy name" />
      ) : (
        <div className="text-sm text-muted-foreground">{attendee.notes ?? 'No notes'}</div>
      )}
      <Button
        size="sm"
        onClick={() =>
          onSave({
            user_id: attendee.user_id,
            status,
            proxy_user_name: status === 'proxy' ? proxyName : null,
            proxy_authorized_by: status === 'proxy' ? currentUserId ?? null : null,
          })
        }
      >
        Save
      </Button>
    </div>
  );
}

function normalizeStatus(status: ActaAttendee['status']): 'present' | 'absent' | 'proxy' | 'excused' {
  if (status === 'present' || status === 'absent' || status === 'proxy' || status === 'excused') {
    return status;
  }
  return 'absent';
}
