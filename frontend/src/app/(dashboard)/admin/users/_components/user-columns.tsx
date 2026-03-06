'use client';

import type { ColumnDef } from '@tanstack/react-table';
import { CheckCircle, XCircle, Lock } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { UserAvatar } from '@/components/shared/user-avatar';
import { StatusBadge } from '@/components/shared/status-badge';
import { RelativeTime } from '@/components/shared/relative-time';
import { selectColumn, dateColumn, actionsColumn } from '@/components/shared/data-table/columns/common-columns';
import { userStatusConfig } from '@/lib/status-configs';
import type { User } from '@/types/models';
import type { RowAction } from '@/types/table';

export interface UserColumnOptions {
  onEdit: (user: User) => void;
  onAssignRoles: (user: User) => void;
  onResetPassword: (user: User) => void;
  onChangeStatus: (user: User) => void;
  onDelete: (user: User) => void;
  onRowClick: (user: User) => void;
  currentUserId: string;
  hasPermission: (p: string) => boolean;
}

export function getUserColumns(options: UserColumnOptions): ColumnDef<User>[] {
  const {
    onEdit,
    onAssignRoles,
    onResetPassword,
    onChangeStatus,
    onDelete,
    currentUserId,
    hasPermission,
  } = options;

  const rowActions = (user: User): RowAction<User>[] => {
    const actions: RowAction<User>[] = [];

    if (hasPermission('users:write')) {
      actions.push({
        label: 'Edit Profile',
        onClick: onEdit,
      });
    }
    if (hasPermission('roles:assign')) {
      actions.push({
        label: 'Assign Roles',
        onClick: onAssignRoles,
      });
    }
    if (hasPermission('users:write')) {
      actions.push({
        label: 'Reset Password',
        onClick: onResetPassword,
      });
    }
    if (hasPermission('users:write')) {
      if (user.status === 'active') {
        actions.push({
          label: 'Suspend User',
          onClick: onChangeStatus,
          variant: 'destructive',
        });
      } else if (user.status === 'suspended') {
        actions.push({
          label: 'Activate User',
          onClick: onChangeStatus,
        });
      }
    }
    if (hasPermission('users:delete')) {
      actions.push({
        label: 'Delete User',
        onClick: onDelete,
        variant: 'destructive',
        hidden: (u) => u.id === currentUserId,
      });
    }

    return actions;
  };

  return [
    selectColumn<User>(),
    {
      id: 'name',
      accessorFn: (row) => `${row.first_name} ${row.last_name}`,
      header: 'User',
      cell: ({ row }) => {
        const user = row.original;
        return (
          <div className="flex items-center gap-3">
            <UserAvatar user={user} size="sm" />
            <div className="min-w-0">
              <p className="font-medium truncate">
                {user.first_name} {user.last_name}
              </p>
              <p className="text-xs text-muted-foreground truncate">{user.email}</p>
            </div>
          </div>
        );
      },
      enableSorting: true,
    },
    {
      id: 'roles',
      header: 'Roles',
      size: 200,
      cell: ({ row }) => {
        const roles = row.original.roles;
        const displayed = roles.slice(0, 2);
        const extra = roles.length - 2;
        return (
          <div className="flex flex-wrap gap-1">
            {displayed.map((role) => (
              <Badge key={role.id} variant="outline" className="text-xs gap-1">
                {role.is_system && <Lock className="h-2.5 w-2.5" />}
                {role.name}
              </Badge>
            ))}
            {extra > 0 && (
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Badge variant="secondary" className="text-xs cursor-default">
                      +{extra} more
                    </Badge>
                  </TooltipTrigger>
                  <TooltipContent>
                    <div className="space-y-1">
                      {roles.slice(2).map((r) => (
                        <p key={r.id} className="text-xs">{r.name}</p>
                      ))}
                    </div>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            )}
          </div>
        );
      },
      enableSorting: false,
    },
    {
      id: 'status',
      accessorKey: 'status',
      header: 'Status',
      size: 120,
      cell: ({ row }) => (
        <StatusBadge
          status={row.original.status}
          config={userStatusConfig}
          variant="dot"
        />
      ),
      enableSorting: true,
    },
    {
      id: 'mfa_enabled',
      accessorKey: 'mfa_enabled',
      header: 'MFA',
      size: 70,
      cell: ({ row }) =>
        row.original.mfa_enabled ? (
          <CheckCircle className="h-4 w-4 text-green-500" aria-label="Enabled" />
        ) : (
          <XCircle className="h-4 w-4 text-muted-foreground" aria-label="Disabled" />
        ),
      enableSorting: true,
    },
    {
      id: 'last_login_at',
      accessorKey: 'last_login_at',
      header: 'Last Login',
      size: 140,
      cell: ({ row }) =>
        row.original.last_login_at ? (
          <RelativeTime date={row.original.last_login_at} />
        ) : (
          <span className="text-muted-foreground text-sm">Never</span>
        ),
      enableSorting: true,
    },
    dateColumn<User>('created_at', 'Created'),
    actionsColumn<User>(rowActions),
  ];
}
