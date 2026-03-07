'use client';

import { SearchInput } from '@/components/shared/forms/search-input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { ActaCommittee } from '@/types/suites';

interface MeetingFiltersProps {
  search: string;
  onSearchChange: (value: string) => void;
  committeeId?: string;
  onCommitteeChange: (value?: string) => void;
  status?: string;
  onStatusChange: (value?: string) => void;
  committees: ActaCommittee[];
  loading?: boolean;
}

export function MeetingFilters({
  search,
  onSearchChange,
  committeeId,
  onCommitteeChange,
  status,
  onStatusChange,
  committees,
  loading = false,
}: MeetingFiltersProps) {
  return (
    <div className="grid gap-3 rounded-xl border bg-card p-4 lg:grid-cols-[1.2fr_0.8fr_0.8fr]">
      <SearchInput
        value={search}
        onChange={onSearchChange}
        placeholder="Search meetings..."
        loading={loading}
      />
      <Select value={committeeId || 'all'} onValueChange={(value) => onCommitteeChange(value === 'all' ? undefined : value)}>
        <SelectTrigger>
          <SelectValue placeholder="All committees" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">All committees</SelectItem>
          {committees.map((committee) => (
            <SelectItem key={committee.id} value={committee.id}>
              {committee.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Select value={status || 'all'} onValueChange={(value) => onStatusChange(value === 'all' ? undefined : value)}>
        <SelectTrigger>
          <SelectValue placeholder="All statuses" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">All statuses</SelectItem>
          <SelectItem value="draft">Draft</SelectItem>
          <SelectItem value="scheduled">Scheduled</SelectItem>
          <SelectItem value="in_progress">In progress</SelectItem>
          <SelectItem value="completed">Completed</SelectItem>
          <SelectItem value="cancelled">Cancelled</SelectItem>
          <SelectItem value="postponed">Postponed</SelectItem>
        </SelectContent>
      </Select>
    </div>
  );
}
