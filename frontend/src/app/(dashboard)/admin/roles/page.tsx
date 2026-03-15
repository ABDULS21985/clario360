"use client";

import { useState } from "react";
import { Plus, Shield, Lock, Key, Edit, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { PageHeader } from "@/components/common/page-header";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { RoleFormDialog } from "./_components/role-form-dialog";
import { useApiQuery } from "@/hooks/use-api";
import api from "@/lib/api";
import type { Role } from "@/types/models";

export default function RoleManagementPage() {
  const [createOpen, setCreateOpen] = useState(false);
  const [editRole, setEditRole] = useState<Role | null>(null);
  const [deleteRole, setDeleteRole] = useState<Role | null>(null);

  const { data, isLoading, error, refetch } = useApiQuery<Role[]>(
    ["roles"],
    "/api/v1/roles"
  );

  const roles = data ?? [];

  if (isLoading) {
    return (
      <div className="space-y-6">
        <PageHeader
          title="Role Management"
          description="Define roles and permissions for your organization"
        />
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-44 rounded-lg" />
          ))}
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-6">
        <PageHeader
          title="Role Management"
          description="Define roles and permissions for your organization"
        />
        <div className="flex flex-col items-center justify-center py-20 gap-3">
          <p className="text-muted-foreground">Failed to load roles.</p>
          <Button variant="outline" onClick={() => refetch()}>
            Retry
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Role Management"
        description="Define roles and permissions for your organization"
        actions={
          <Button onClick={() => setCreateOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Create Role
          </Button>
        }
      />

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {roles.map((role) => (
          <Card key={role.id} className="flex flex-col">
            <CardHeader className="pb-3">
              <div className="flex items-start justify-between gap-2">
                <div className="flex items-center gap-2 min-w-0">
                  {role.is_system ? (
                    <Lock className="h-4 w-4 text-muted-foreground shrink-0" />
                  ) : (
                    <Shield className="h-4 w-4 text-primary shrink-0" />
                  )}
                  <CardTitle className="text-base truncate">{role.name}</CardTitle>
                </div>
                {role.is_system && (
                  <Badge variant="outline" className="shrink-0 text-xs">
                    System
                  </Badge>
                )}
              </div>
              {role.description && (
                <p className="text-xs text-muted-foreground line-clamp-2">
                  {role.description}
                </p>
              )}
            </CardHeader>
            <CardContent className="flex-1">
              <div className="flex items-center gap-4 text-sm text-muted-foreground mb-4">
                <span className="flex items-center gap-1">
                  <Key className="h-3.5 w-3.5" />
                  {role.permissions.length} permissions
                </span>
              </div>
              <div className="flex gap-2">
                {role.is_system ? (
                  <Button size="sm" variant="outline" className="flex-1" disabled>
                    <Lock className="mr-2 h-3.5 w-3.5" />
                    View
                  </Button>
                ) : (
                  <>
                    <Button
                      size="sm"
                      variant="outline"
                      className="flex-1"
                      onClick={() => setEditRole(role)}
                    >
                      <Edit className="mr-2 h-3.5 w-3.5" />
                      Edit
                    </Button>
                    <Button
                      size="sm"
                      variant="destructive"
                      onClick={() => setDeleteRole(role)}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                  </>
                )}
              </div>
            </CardContent>
          </Card>
        ))}

        {/* Create new role card */}
        <button
          className="flex flex-col items-center justify-center rounded-lg border-2 border-dashed border-muted-foreground/25 p-8 hover:border-muted-foreground/50 hover:bg-muted/30 transition-colors text-muted-foreground gap-2 min-h-[11rem]"
          onClick={() => setCreateOpen(true)}
        >
          <Plus className="h-6 w-6" />
          <span className="text-sm font-medium">Create New Role</span>
        </button>
      </div>

      <RoleFormDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onSuccess={refetch}
      />

      {editRole && (
        <RoleFormDialog
          role={editRole}
          open={!!editRole}
          onOpenChange={(o) => !o && setEditRole(null)}
          onSuccess={refetch}
        />
      )}

      {deleteRole && (
        <ConfirmDialog
          open={!!deleteRole}
          onOpenChange={(o) => !o && setDeleteRole(null)}
          title="Delete Role"
          description={`Delete "${deleteRole.name}"? Users with this role will lose associated permissions.`}
          confirmLabel="Delete"
          variant="destructive"
          onConfirm={async () => {
            await api.delete(`/api/v1/roles/${deleteRole.id}`);
            toast.success("Role deleted");
            refetch();
          }}
        />
      )}
    </div>
  );
}
