'use client';

import { Lock, FileEdit, Users, Settings, Eye, Pencil, Trash2 } from 'lucide-react';
import { Card, CardContent, CardFooter } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';
import type { Role } from '@/types/models';

interface RoleCardProps {
  role: Role;
  userCount: number | undefined;
  onView: () => void;
  onEdit: () => void;
  onDelete: () => void;
  canEdit: boolean;
  canDelete: boolean;
}

export function RoleCard({
  role,
  userCount,
  onView,
  onEdit,
  onDelete,
  canEdit,
  canDelete,
}: RoleCardProps) {
  return (
    <Card
      className={cn(
        'flex flex-col transition-shadow hover:shadow-md cursor-pointer',
        role.is_system && 'border-primary/20'
      )}
      onClick={onView}
    >
      <CardContent className="flex-1 pt-5 pb-3">
        <div className="flex items-start gap-3">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-muted">
            {role.is_system ? (
              <Lock className="h-4 w-4 text-muted-foreground" />
            ) : (
              <FileEdit className="h-4 w-4 text-muted-foreground" />
            )}
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <h3 className="font-semibold text-sm truncate">{role.name}</h3>
              <Badge
                variant={role.is_system ? 'default' : 'outline'}
                className="text-xs shrink-0"
              >
                {role.is_system ? 'System' : 'Custom'}
              </Badge>
            </div>
            {role.description && (
              <p className="text-xs text-muted-foreground mt-1 line-clamp-2">
                {role.description}
              </p>
            )}
          </div>
        </div>

        <div className="mt-3 flex items-center gap-3 text-xs text-muted-foreground">
          <span className="flex items-center gap-1">
            <Users className="h-3 w-3" />
            {userCount === undefined ? (
              <Skeleton className="h-3 w-6 inline-block" />
            ) : (
              `${userCount} users`
            )}
          </span>
          <span className="flex items-center gap-1">
            <Settings className="h-3 w-3" />
            {role.permissions.length} perms
          </span>
        </div>
      </CardContent>

      <CardFooter
        className="border-t pt-3 pb-3 gap-2"
        onClick={(e) => e.stopPropagation()}
      >
        <Button
          variant="ghost"
          size="sm"
          className="flex-1 h-7 text-xs"
          onClick={onView}
        >
          <Eye className="mr-1.5 h-3.5 w-3.5" />
          {role.is_system ? 'View' : 'Details'}
        </Button>
        {!role.is_system && canEdit && (
          <Button
            variant="ghost"
            size="sm"
            className="flex-1 h-7 text-xs"
            onClick={onEdit}
          >
            <Pencil className="mr-1.5 h-3.5 w-3.5" />
            Edit
          </Button>
        )}
        {!role.is_system && canDelete && (
          <Button
            variant="ghost"
            size="sm"
            className="flex-1 h-7 text-xs text-destructive hover:text-destructive"
            onClick={onDelete}
          >
            <Trash2 className="mr-1.5 h-3.5 w-3.5" />
            Delete
          </Button>
        )}
      </CardFooter>
    </Card>
  );
}

export function RoleCardSkeleton() {
  return (
    <Card className="flex flex-col">
      <CardContent className="flex-1 pt-5 pb-3">
        <div className="flex items-start gap-3">
          <Skeleton className="h-9 w-9 rounded-full shrink-0" />
          <div className="flex-1 space-y-2">
            <Skeleton className="h-4 w-3/4" />
            <Skeleton className="h-3 w-full" />
          </div>
        </div>
        <div className="mt-3 flex gap-3">
          <Skeleton className="h-3 w-16" />
          <Skeleton className="h-3 w-16" />
        </div>
      </CardContent>
      <CardFooter className="border-t pt-3 pb-3">
        <Skeleton className="h-7 w-full" />
      </CardFooter>
    </Card>
  );
}
