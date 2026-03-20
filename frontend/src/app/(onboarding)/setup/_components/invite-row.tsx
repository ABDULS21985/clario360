'use client';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

import type { InvitationDraft, RoleRecord } from './shared';

export function InviteRow({
  index,
  row,
  roles,
  canRemove,
  onChange,
  onRemove,
}: {
  index: number;
  row: InvitationDraft;
  roles: RoleRecord[];
  canRemove: boolean;
  onChange: (index: number, field: keyof InvitationDraft, value: string) => void;
  onRemove: (index: number) => void;
}) {
  return (
    <div className="grid gap-3 rounded-2xl border border-slate-200 bg-white p-4 md:grid-cols-[1.5fr_1fr_auto]">
      <div className="space-y-2">
        <Label>Email</Label>
        <Input
          type="email"
          value={row.email}
          onChange={(event) => onChange(index, 'email', event.target.value)}
          placeholder="alice@company.com"
        />
      </div>
      <div className="space-y-2">
        <Label>Role</Label>
        <select
          value={row.role_slug}
          onChange={(event) => onChange(index, 'role_slug', event.target.value)}
          className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
        >
          {roles.map((role) => (
            <option key={role.id} value={role.slug}>
              {role.name}
            </option>
          ))}
        </select>
      </div>
      <div className="flex items-end">
        <Button type="button" variant="ghost" onClick={() => onRemove(index)} disabled={!canRemove}>
          Remove
        </Button>
      </div>
    </div>
  );
}
