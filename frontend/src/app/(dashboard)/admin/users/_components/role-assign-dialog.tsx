"use client";

import { useState, useEffect } from "react";
import { Lock } from "lucide-react";
import { toast } from "sonner";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { useApiQuery } from "@/hooks/use-api";
import api from "@/lib/api";
import type { User, Role } from "@/types/models";

interface RoleAssignDialogProps {
  user: User;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess: () => void;
}

export function RoleAssignDialog({
  user,
  open,
  onOpenChange,
  onSuccess,
}: RoleAssignDialogProps) {
  const [selectedRoleIds, setSelectedRoleIds] = useState<Set<string>>(
    new Set(user.roles.map((r) => r.id))
  );
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setSelectedRoleIds(new Set(user.roles.map((r) => r.id)));
  }, [user]);

  const { data: rolesData, isLoading } = useApiQuery<{ data: Role[] }>(
    ["roles"],
    "/api/v1/roles",
    { enabled: open }
  );

  const allRoles = rolesData?.data ?? [];

  const toggle = (roleId: string) => {
    setSelectedRoleIds((prev) => {
      const next = new Set(prev);
      if (next.has(roleId)) next.delete(roleId);
      else next.add(roleId);
      return next;
    });
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      const currentIds = new Set(user.roles.map((r) => r.id));
      const toAdd = [...selectedRoleIds].filter((id) => !currentIds.has(id));
      const toRemove = [...currentIds].filter((id) => !selectedRoleIds.has(id));

      await Promise.all([
        ...toAdd.map((id) =>
          api.post(`/api/v1/users/${user.id}/roles`, { role_id: id })
        ),
        ...toRemove.map((id) =>
          api.delete(`/api/v1/users/${user.id}/roles/${id}`)
        ),
      ]);

      toast.success("Roles updated successfully");
      onOpenChange(false);
      onSuccess();
    } catch {
      toast.error("Failed to update roles");
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Assign Roles</DialogTitle>
          <DialogDescription>
            Select roles for {user.first_name} {user.last_name}.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-2 max-h-80 overflow-y-auto">
          {isLoading ? (
            Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-12 rounded" />
            ))
          ) : allRoles.length === 0 ? (
            <p className="text-sm text-muted-foreground text-center py-4">
              No roles available.
            </p>
          ) : (
            allRoles.map((role) => (
              <div
                key={role.id}
                className="flex items-start gap-3 rounded-lg border border-border p-3 hover:bg-muted/50 transition-colors"
              >
                <Checkbox
                  id={`role-${role.id}`}
                  checked={selectedRoleIds.has(role.id)}
                  onCheckedChange={() => toggle(role.id)}
                  disabled={saving}
                />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <Label
                      htmlFor={`role-${role.id}`}
                      className="font-medium text-sm cursor-pointer"
                    >
                      {role.name}
                    </Label>
                    {role.is_system && (
                      <Badge variant="outline" className="gap-1 text-xs">
                        <Lock className="h-2.5 w-2.5" />
                        System
                      </Badge>
                    )}
                  </div>
                  {role.description && (
                    <p className="text-xs text-muted-foreground mt-0.5">
                      {role.description}
                    </p>
                  )}
                  <p className="text-xs text-muted-foreground mt-0.5">
                    {role.permissions.length} permissions
                  </p>
                </div>
              </div>
            ))
          )}
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={saving}
          >
            Cancel
          </Button>
          <Button onClick={handleSave} disabled={saving || isLoading}>
            {saving ? "Saving..." : "Save Changes"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
