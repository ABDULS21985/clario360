'use client';

import { useQuery } from '@tanstack/react-query';
import { Lock, FileEdit, Users } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Skeleton } from '@/components/ui/skeleton';
import { apiGet } from '@/lib/api';
import type { PaginatedResponse } from '@/types/api';
import type { Role, User } from '@/types/models';

interface RoleDetailDialogProps {
  role: Role;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onEdit?: () => void;
}

export function RoleDetailDialog({
  role,
  open,
  onOpenChange,
  onEdit,
}: RoleDetailDialogProps) {
  const { data: usersData, isLoading: usersLoading } = useQuery({
    queryKey: ['role-users', role.id],
    queryFn: () =>
      apiGet<PaginatedResponse<User>>(`/api/v1/roles/${role.id}/users`, {
        per_page: 10,
      }),
    enabled: open,
  });

  const assignedUsers = usersData?.data ?? [];
  const totalUsers = usersData?.meta.total ?? 0;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-muted">
              {role.is_system ? (
                <Lock className="h-5 w-5 text-muted-foreground" />
              ) : (
                <FileEdit className="h-5 w-5 text-muted-foreground" />
              )}
            </div>
            <div>
              <DialogTitle>{role.name}</DialogTitle>
              <DialogDescription className="mt-0.5">
                {role.is_system ? 'System role' : 'Custom role'} · slug:{' '}
                <span className="font-mono text-xs">{role.slug}</span>
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <ScrollArea className="max-h-96">
          <div className="space-y-4 pr-2">
            {role.description && (
              <p className="text-sm text-muted-foreground">{role.description}</p>
            )}

            <Separator />

            <div className="space-y-2">
              <h4 className="text-sm font-medium">
                Permissions ({role.permissions.length})
              </h4>
              <div className="flex flex-wrap gap-1.5">
                {role.permissions.map((p) => (
                  <Badge key={p} variant="outline" className="text-xs font-mono">
                    {p}
                  </Badge>
                ))}
                {role.permissions.length === 0 && (
                  <p className="text-sm text-muted-foreground">No permissions assigned.</p>
                )}
              </div>
            </div>

            <Separator />

            <div className="space-y-2">
              <h4 className="text-sm font-medium">
                Assigned Users ({usersLoading ? '...' : totalUsers})
              </h4>
              {usersLoading ? (
                <div className="space-y-2">
                  {Array.from({ length: 3 }).map((_, i) => (
                    <Skeleton key={i} className="h-8 w-full" />
                  ))}
                </div>
              ) : assignedUsers.length === 0 ? (
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Users className="h-4 w-4" />
                  No users assigned to this role.
                </div>
              ) : (
                <div className="space-y-1">
                  {assignedUsers.map((user) => (
                    <div
                      key={user.id}
                      className="flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-muted/40"
                    >
                      <div className="flex h-6 w-6 items-center justify-center rounded-full bg-primary/10 text-xs font-medium text-primary">
                        {user.first_name[0]}{user.last_name[0]}
                      </div>
                      <div className="min-w-0 flex-1">
                        <p className="text-sm font-medium truncate">
                          {user.first_name} {user.last_name}
                        </p>
                        <p className="text-xs text-muted-foreground truncate">{user.email}</p>
                      </div>
                    </div>
                  ))}
                  {totalUsers > 10 && (
                    <p className="text-xs text-muted-foreground px-2">
                      +{totalUsers - 10} more users
                    </p>
                  )}
                </div>
              )}
            </div>
          </div>
        </ScrollArea>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
          {!role.is_system && onEdit && (
            <Button onClick={onEdit}>
              Edit Role
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
